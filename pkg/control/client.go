package control

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
)

func MkdirAll(path string) (string, error) {
	cli := getClient()

	data := new(bytes.Buffer)
	if err := json.NewEncoder(data).Encode(MakeDirectoryRequest{
		Path: path,
	}); err != nil {
		return path, err
	}

	resp, err := cli.Post("http://control/mkdir", "application/json", data)
	if err != nil {
		return path, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return path, err
	}

	decoded := MakeDirectoryResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return path, err
	}

	return decoded.CreatedPath, nil
}

func CheckDevice(path string) (bool, error) {
	cli := getClient()

	data := new(bytes.Buffer)
	if err := json.NewEncoder(data).Encode(CheckDeviceRequest{
		Path: path,
	}); err != nil {
		return false, err
	}

	resp, err := cli.Post("http://control/checkDevice", "application/json", data)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, err
	}

	decoded := CheckDeviceResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return false, err
	}

	return decoded.Exists, nil
}

func RunCommand(containerId string, command string) (bool, error) {
	cli := getClient()

	data := new(bytes.Buffer)
	if err := json.NewEncoder(data).Encode(RunCommandRequest{
		ContainerId: containerId,
		Command:     command,
	}); err != nil {
		return false, err
	}

	resp, err := cli.Post("http://control/runCommand", "application/json", data)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, err
	}

	decoded := RunCommandResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return false, err
	}

	return decoded.ExitCode == 0, nil // TODO exit code
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
