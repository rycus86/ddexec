package exec

import (
	"os"
	"os/exec"
)

func prepareXauth() error {
	target := getXauth()

	if fi, err := os.Stat(target); err == nil && !fi.IsDir() && fi.Size() > 0 {
		return nil // already exists
	}

	// TODO this won't work without xauth and a shell, but that's probably OK?
	return exec.Command("sh", "-c",
		"touch "+target+" && "+
			"xauth nlist "+os.Getenv("DISPLAY")+" | sed -e 's/^..../ffff/' | xauth -f "+target+" nmerge -",
	).Run()
}

func getXauth() string {
	s := os.Getenv("XAUTH")
	if s == "" {
		s = "/tmp/.docker.xauth"
	}
	return s
}
