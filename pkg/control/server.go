package control

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/rycus86/ddexec/pkg/debug"
	"github.com/rycus86/ddexec/pkg/env"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path"
)

const EnvServerSocket = "DDEXEC_SERVER_SOCK"

var serverSocket string

func GetServerSocket() string {
	sock := os.Getenv(EnvServerSocket)
	if sock == "" {
		sock = serverSocket
	}
	return sock
}

func GetDirectoryToShare() string {
	if sock := GetServerSocket(); sock == "" {
		panic(errors.New("no control socket found"))
	} else {
		return path.Dir(sock)
	}
}

func StartServerIfNecessary() {
	if env.IsSet(EnvServerSocket) {
		return
	}

	tmpDir, err := ioutil.TempDir("", "ddexec")
	if err != nil {
		panic(err)
	}

	serverSocket = path.Join(tmpDir, "ddexec.sock")

	l, err := net.Listen("unix", serverSocket)
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/mkdir", handleMkdir)

	go runServer(l, tmpDir)
}

func runServer(l net.Listener, tmpDir string) {
	defer os.RemoveAll(tmpDir)
	defer l.Close()

	http.Serve(l, nil)
}

func handleMkdir(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if r.Method != "POST" {
		w.WriteHeader(400)
		return
	}

	rawPath, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(400)
		return
	}

	path := Source(string(rawPath))

	if fi, err := os.Stat(path); err == nil {
		if fi.IsDir() {
			w.WriteHeader(200)
			w.Header().Add("Content-Type", "text/plain") // TODO use JSON
			w.Write([]byte(path))
		} else {
			w.WriteHeader(409)
		}
		return
	}

	if debug.IsEnabled() {
		fmt.Println("Creating directory for child at:", path)
	}

	defer func() {
		if e := recover(); e != nil {
			w.WriteHeader(500)
		}
	}()

	created := EnsureSourceExists(path)

	w.WriteHeader(200)
	w.Header().Add("Content-Type", "text/plain") // TODO use JSON
	w.Write([]byte(created))
}
