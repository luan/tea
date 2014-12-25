package handlers

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"io"
	"net/http"
	"os"
	"os/exec"
	"time"
	"unicode/utf8"

	"github.com/gorilla/websocket"
	"github.com/kr/pty"
	"github.com/luan/tea/utils"
	"github.com/pivotal-golang/lager"
)

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

	done := make(chan bool)
	quit := make(chan struct{})
	go s.readLoop(ws, f, done, quit)
	go s.writeLoop(ws, f, done, quit)
	go func() {
		pingInterval := 5 * time.Second
		c := time.Tick(pingInterval)
		for _ = range c {
			select {
			case <-quit:
				return
			default:
				if err := ws.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(2*time.Second)); err != nil {
					done <- true
				}
			}
		}
		ws.SetPongHandler(func(string) error {
			return ws.SetReadDeadline(time.Now().Add(pingInterval + 500*time.Millisecond))
		})
	}()
	<-done
	close(quit)

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

func (s *ShellHandler) readLoop(c *websocket.Conn, w *os.File, done chan<- bool, quit <-chan struct{}) {
	log := s.logger.Session("shell-handler")
	for {
		select {
		case <-quit:
			return
		default:
			mType, m, err := c.ReadMessage()
			if mType == websocket.TextMessage {
				if err != nil {
					log.Error("read-error", err)
					done <- true
					return
				}
				w.Write(m)
			} else if mType == websocket.BinaryMessage {
				dec := gob.NewDecoder(bytes.NewReader(m))
				winsize := &utils.Winsize{}
				dec.Decode(winsize)
				utils.SetWinsize(w.Fd(), winsize)
				if err != nil {
					log.Error("read-error", err)
					done <- true
					return
				}
			}
		}
	}
}

func (s *ShellHandler) writeLoop(c *websocket.Conn, r io.Reader, done chan<- bool, quit <-chan struct{}) {
	log := s.logger.Session("shell-handler")
	br := bufio.NewReader(r)
	for {
		select {
		case <-quit:
			return
		default:
			x, size, err := br.ReadRune()
			if err != nil {
				log.Error("write-error", err)
				done <- true
				return
			}

			p := make([]byte, size)
			utf8.EncodeRune(p, x)

			err = c.WriteMessage(websocket.TextMessage, p)
			if err != nil {
				log.Error("write-error", err)
				done <- true
				return
			}
		}
	}
}
