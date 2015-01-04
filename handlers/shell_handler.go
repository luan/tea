package handlers

import (
	"bufio"
	"io"
	"net/http"
	"os"
	"os/exec"
	"time"
	"unicode/utf8"

	"github.com/gorilla/websocket"
	"github.com/kr/pty"
	"github.com/pivotal-golang/lager"
)

type ioItem struct {
	data []byte
	err  error
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type ShellHandler struct {
	logger lager.Logger
}

func NewShellHandler(logger lager.Logger) *ShellHandler {
	return &ShellHandler{
		logger: logger,
	}
}

func (s *ShellHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log := s.logger.Session("shell-handler")
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("websocket-upgrade-failed", err)
		return
	}
	defer ws.Close()

	shell := "/bin/bash"
	log.Info("starting-shell", lager.Data{"shell": shell})
	cmd := exec.Command(shell)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "TERM=xterm-256color")
	f, err := pty.Start(cmd)
	if err != nil {
		log.Error("error-starting-pty", err)
		return
	}

	input, output := make(chan ioItem), make(chan ioItem)
	quit := make(chan struct{})
	go readLoop(ws, input, quit)
	go writeLoop(f, output, quit)

	pingInterval := 5 * time.Second
	ws.SetPongHandler(func(string) error {
		return ws.SetReadDeadline(time.Now().Add(pingInterval + 500*time.Millisecond))
	})
	var pong chan error
dance:
	for {
		ping := time.After(pingInterval)

		select {
		case i := <-input:
			if i.err != nil {
				log.Error("socket-read-error", err)
				close(quit)
			}
			_, err = f.Write(i.data)
			if i.err != nil {
				log.Error("read-error", err)
				close(quit)
			}
		case o := <-output:
			if o.err != nil {
				log.Error("write-error", err)
				close(quit)
			}
			err = ws.WriteMessage(websocket.TextMessage, o.data)
			if err != nil {
				log.Error("socket-write-error", err)
				close(quit)
			}
		case <-ping:
			pong = make(chan error)
			go func() {
				pong <- ws.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(pingInterval))
			}()
		case pongerr := <-pong:
			pong = nil
			if pongerr != nil {
				log.Error("ping-failed", pongerr)
				close(quit)
			}
		case <-quit:
			break dance
		}
	}

	log.Info("closing-shell")
	err = f.Close()
	if err != nil {
		log.Error("closing-shell-failed", err)
	}

	err = ws.Close()
	if err != nil {
		log.Error("closing-websocket-failed", err)
	}
}

func readLoop(c *websocket.Conn, input chan<- ioItem, quit <-chan struct{}) {
	for {
		select {
		case <-quit:
			return
		default:
			mType, m, err := c.ReadMessage()
			if mType == websocket.TextMessage {
				if err != nil {
					input <- ioItem{nil, err}
					return
				}
				input <- ioItem{m, nil}
			}
		}
	}
}

func writeLoop(r io.Reader, output chan<- ioItem, quit <-chan struct{}) {
	br := bufio.NewReader(r)
	for {
		select {
		case <-quit:
			return
		default:
			x, size, err := br.ReadRune()
			if err != nil {
				output <- ioItem{nil, err}
				return
			}
			p := make([]byte, size)
			utf8.EncodeRune(p, x)
			output <- ioItem{p, nil}
		}
	}
}
