package handlers

import (
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"

	"github.com/kr/pty"
	"github.com/pivotal-golang/lager"
)

type SSHHandler struct {
	logger lager.Logger
}

func NewSSHHandler(logger lager.Logger) *SSHHandler {
	return &SSHHandler{
		logger: logger,
	}
}

func (s *SSHHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log := s.logger.Session("ssh-handler")
	log.Info("starting-ssh")
	w.WriteHeader(http.StatusOK)
	client, _, err := w.(http.Hijacker).Hijack()
	if err != nil {
		log.Error("cannot-hijack", err)
		return
	}
	defer client.Close()
	log.Info("hijacked", lager.Data{"client": client.RemoteAddr()})

	server, err := net.Dial("tcp", "127.0.0.1:22000")
	log.Info("server", lager.Data{"server": server.RemoteAddr()})
	if err != nil {
		log.Error("connecting-to-ssh", err)
		return
	}
	forwardIO(client, server)
	log.Info("shell-closed")
}

func forwardIO(a, b net.Conn) {
	done := make(chan bool, 2)

	fwd := func(dst io.Writer, src io.Reader) {
		io.Copy(dst, src)
		done <- true
	}

	go fwd(a, b)
	go fwd(b, a)

	<-done
}

func startShell(shell string) (*os.File, error) {
	cmd := exec.Command(shell)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "TERM=xterm-256color")
	f, err := pty.Start(cmd)
	if err != nil {
		return nil, err
	}
	return f, nil
}
