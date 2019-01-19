package control

import (
	"context"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
)

func MkdirAll(path string) (string, error) {
	cli := getClient()

	resp, err := cli.Post("http://control/mkdir", "text/plain", strings.NewReader(path))
	if err != nil {
		return path, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return path, err
	}

	created, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return path, err
	}

	return string(created), nil
}

func getClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", GetServerSocket())
			},
		},
	}
}
