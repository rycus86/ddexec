package control

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/rycus86/ddexec/pkg/debug"
	"github.com/rycus86/ddexec/pkg/env"
	"github.com/rycus86/ddexec/pkg/xdgexec"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path"
)

const (
	EnvServerSocket = "DDEXEC_SERVER_SOCK"
	EnvControlDir   = "DDEXEC_CONTROL_DIR"
)

var serverSocket string

func GetServerSocket() string {
	sock := os.Getenv(EnvServerSocket)
	if sock == "" {
		sock = serverSocket
	}
	return sock
}

func GetDirectoryToShare() string {
	if env.IsSet(EnvControlDir) {
		return os.Getenv(EnvControlDir)
	} else if sock := GetServerSocket(); sock == "" {
		panic(errors.New("no control socket found"))
	} else {
		return path.Dir(sock)
	}
}

func StartServerIfNecessary() {
	if env.IsSet(EnvServerSocket) || env.IsSet(EnvControlDir) {
		return
	}

	var controlDir string

	if env.IsSet(EnvControlDir) {
		controlDir = EnsureSourceExists(os.Getenv(EnvControlDir))

	} else {
		tmpDir, err := ioutil.TempDir("", "ddexec")
		if err != nil {
			panic(err)
		}

		controlDir = tmpDir
	}

	serverSocket = path.Join(controlDir, "ddexec.sock")

	l, err := net.Listen("unix", serverSocket)
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/mkdir", handleMkdir)
	http.HandleFunc("/checkDevice", handleCheckDevice)
	http.HandleFunc("/runCommand", handleRunCommand)

	go runServer(l, controlDir)
}

func runServer(l net.Listener, controlDir string) {
	defer os.RemoveAll(controlDir)
	defer l.Close()

	http.Serve(l, nil)
}

func handleMkdir(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if r.Method != "POST" {
		w.WriteHeader(400)
		return
	}

	request := MakeDirectoryRequest{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		w.WriteHeader(400)
		return
	}

	targetPath := Source(string(request.Path))

	if fi, err := os.Stat(targetPath); err == nil {
		if fi.IsDir() {
			w.WriteHeader(200)
			w.Header().Add("Content-Type", "application/json")
			json.NewEncoder(w).Encode(&MakeDirectoryResponse{
				CreatedPath: targetPath,
			})
		} else {
			w.WriteHeader(409)
		}
		return
	}

	if debug.IsEnabled() {
		fmt.Println("Creating directory for child at:", targetPath)
	}

	defer func() {
		if e := recover(); e != nil {
			w.WriteHeader(500)
		}
	}()

	created := EnsureSourceExists(targetPath)

	w.WriteHeader(200)
	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(&MakeDirectoryResponse{
		CreatedPath: created,
	})
}

func handleCheckDevice(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if r.Method != "POST" {
		w.WriteHeader(400)
		return
	}

	request := CheckDeviceRequest{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		w.WriteHeader(400)
		return
	}

	_, err := os.Stat(request.Path)

	w.WriteHeader(200)
	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(&CheckDeviceResponse{
		Exists: err == nil,
	})
}

func handleRunCommand(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if r.Method != "POST" {
		w.WriteHeader(400)
		return
	}

	request := RunCommandRequest{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		w.WriteHeader(400)
		return
	}

	var exitCode int
	if ok, err := xdgexec.ExecInContainer(request.ContainerId, request.Command); err == nil && ok {
		exitCode = 0
	} else {
		exitCode = 1 // TODO proxy the actual exit code, maybe logs too
	}

	w.WriteHeader(200)
	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(&RunCommandResponse{
		ExitCode: exitCode,
	})
}
