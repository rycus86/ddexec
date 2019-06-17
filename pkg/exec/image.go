package exec

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/term"
	"github.com/pkg/errors"
	"github.com/rycus86/ddexec/pkg/config"
	"github.com/rycus86/ddexec/pkg/debug"
	"github.com/rycus86/ddexec/pkg/env"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

func prepareAndProcessImage(cli *client.Client, c *config.AppConfiguration, sc *config.StartupConfiguration) {
	var (
		image             types.ImageInspect
		shouldBuildOrPull = shouldPullImage()

		err error
	)

	if !shouldBuildOrPull {
		image, _, err = cli.ImageInspectWithRaw(context.TODO(), c.Image)
		if err != nil {
			if client.IsErrNotFound(err) {
				shouldBuildOrPull = true
			} else {
				panic(err)
			}
		}
	}

	if shouldBuildOrPull {
		if c.Dockerfile != "" {
			buildImage(cli, c)
		} else {
			if debug.IsEnabled() {
				fmt.Println("Pulling image for", c.Image, "...")
			}

			if reader, err := cli.ImagePull(
				context.TODO(),
				c.Image, // TODO maybe allow having the image name empty and default to the filename
				types.ImagePullOptions{}); err != nil {
				panic(err)
			} else {
				defer reader.Close()

				var pullMessage jsonmessage.JSONMessage
				for {
					if err := json.NewDecoder(reader).Decode(&pullMessage); err != nil {
						break // TODO probably should check if this was EOF or something
					}

					if pullMessage.Error != nil {
						panic(pullMessage.Error.Error())
					} else if debug.IsEnabled() {
						_, isTerminal := term.GetFdInfo(os.Stdout)
						pullMessage.Display(os.Stdout, isTerminal)
					}
				}
			}
		}

		if image, _, err = cli.ImageInspectWithRaw(context.TODO(), c.Image); err != nil {
			panic(err)
		}
	}

	if c.Dockerfile != "" {
		hash := hashDockerfile(c.Dockerfile)

		if prevHash, ok := image.Config.Labels["com.github.rycus86.ddexec.dockerfile.hash"]; ok && hash == prevHash {
			// OK, we're up to date
		} else {
			buildImage(cli, c)

			if image, _, err = cli.ImageInspectWithRaw(context.TODO(), c.Image); err != nil {
				panic(err)
			} else if image.Config.Labels["com.github.rycus86.ddexec.dockerfile.hash"] != hash {
				panic(errors.New("the new image hash does not match the Dockerfile contents"))
			}
		}
	}

	sc.ImageUser = image.Config.User

	for _, item := range image.Config.Env {
		if strings.HasPrefix(item, "PATH=") {
			sc.EnvPath = strings.TrimPrefix(item, "PATH=")
			break
		} else if strings.HasPrefix(item, "HOME=") {
			sc.ImageHome = strings.TrimPrefix(item, "HOME=")
			break
		}
	}
}

func buildImage(cli *client.Client, c *config.AppConfiguration) {
	if debug.IsEnabled() {
		fmt.Println("Building image for", c.Image, "...")
	}

	bctx := prepareBuildContext(c)

	if response, err := cli.ImageBuild(context.TODO(), bctx, types.ImageBuildOptions{
		Labels: map[string]string{
			"com.github.rycus86.ddexec.built_at":        time.Now().Format(time.RFC3339),
			"com.github.rycus86.ddexec.dockerfile.hash": hashDockerfile(c.Dockerfile), // TODO const label key
		},
		Tags:        []string{c.Image}, // TODO infer image name from filename if empty?
		Remove:      true,
		ForceRemove: true,
		PullParent:  shouldPullImage(),
		NoCache:     shouldSkipBuildCache(),
	}); err != nil {
		panic(err)
	} else {
		defer response.Body.Close()

		var buildMessage jsonmessage.JSONMessage
		for {
			if err := json.NewDecoder(response.Body).Decode(&buildMessage); err != nil {
				if err == io.EOF {
					break
				} else if debug.IsEnabled() {
					fmt.Printf("Error reading the build output : %s (%T)\n", err, err)
				}
			}

			if buildMessage.Error != nil {
				panic(buildMessage.Error.Error())
			} else if debug.IsEnabled() {
				_, isTerminal := term.GetFdInfo(os.Stdout)
				buildMessage.Display(os.Stdout, isTerminal)
			}
		}
	}
}

func prepareBuildContext(c *config.AppConfiguration) io.Reader {
	target, err := ioutil.TempFile("", "ddexec*.Dockerfile")
	if err != nil {
		panic(err)
	}
	defer os.Remove(target.Name())

	if debug.IsEnabled() {
		fmt.Println("Dockerfile:")
		fmt.Println(c.Dockerfile)
	}

	target.WriteString(c.Dockerfile)
	target.Close()

	if tar, err := createTar( // TODO is this generic enough, is it at the right place?
		fileToCopy{Source: target.Name(), Target: "/Dockerfile"},
	); err != nil {
		panic(err)
	} else {
		return tar
	}
}

func shouldPullImage() bool {
	return env.IsSet("DDEXEC_PULL") || env.IsSet("DDEXEC_REBUILD")
}

func shouldSkipBuildCache() bool {
	return env.IsSet("DDEXEC_NO_CACHE") || env.IsSet("DDEXEC_REBUILD")
}

func hashDockerfile(dockerfile string) string {
	h := md5.New()
	io.WriteString(h, dockerfile)
	return hex.EncodeToString(h.Sum(nil))
}
