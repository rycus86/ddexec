package exec

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/rycus86/ddexec/pkg/config"
	"io/ioutil"
	"strings"
)

func prepareAndProcessImage(cli *client.Client, c *config.Configuration, sc *config.StartupConfiguration) {
	image, _, err := cli.ImageInspectWithRaw(context.TODO(), c.Image)
	if err != nil {
		if client.IsErrNotFound(err) { // TODO we might want to build here
			if reader, err := cli.ImagePull(
				context.TODO(),
				c.Image, // TODO maybe allow having the image name empty and default to the filename
				types.ImagePullOptions{}); err != nil {
				panic(err)
			} else {
				defer reader.Close()
				ioutil.ReadAll(reader) // TODO is there anything to do with this?
			}
		} else {
			panic(err)
		}
	} // TODO check here if we want to update the image

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
