package exec

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/rycus86/ddexec/pkg/config"
	"github.com/rycus86/ddexec/pkg/debug"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

func prepareAndProcessImage(cli *client.Client, c *config.AppConfiguration, sc *config.StartupConfiguration) {
	image, _, err := cli.ImageInspectWithRaw(context.TODO(), c.Image)
	if err != nil {
		if client.IsErrNotFound(err) { // TODO we might want to build here
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
					ioutil.ReadAll(reader) // TODO is there anything to do with this?
				}
			}

			if image, _, err = cli.ImageInspectWithRaw(context.TODO(), c.Image); err != nil {
				panic(err)
			}
		} else {
			panic(err)
		}
	} // TODO check here if we want to update the image

	if c.Dockerfile != "" {
		hash := hashDockerfile(c.Dockerfile)

		if prevHash, ok := image.Config.Labels["ddexec.source.dockerfile.hash"]; ok && hash == prevHash {
			// OK, we're up to date
		} else {
			buildImage(cli, c)

			if image, _, err = cli.ImageInspectWithRaw(context.TODO(), c.Image); err != nil {
				panic(err)
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
			"ddexec.source.dockerfile.hash": hashDockerfile(c.Dockerfile), // TODO const label key
		},
		Tags: []string{c.Image}, // TODO infer image name from filename if empty?
		Remove: true,
	}); err != nil {
		panic(err)
	} else {
		defer response.Body.Close()
		ioutil.ReadAll(response.Body) // TODO is there anything to do with this?
	}
}

func prepareBuildContext(c *config.AppConfiguration) io.Reader {
	target, err := ioutil.TempFile("", "ddexec*.Dockerfile")
	if err != nil {
		panic(err)
	}
	defer os.Remove(target.Name())

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

func hashDockerfile(dockerfile string) string {
	h := md5.New()
	io.WriteString(h, dockerfile)
	return hex.EncodeToString(h.Sum(nil))
}
