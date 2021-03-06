package handlers

import (
	"bytes"
	"encoding/gob"
	"io"
	"net/http"

	"github.com/gorilla/websocket"
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
	cmdPipe, err := startShell(shell)
	if err != nil {
		log.Error("error-starting-pty", err)
		return
	}

	errc := make(chan error, 2)
	wsPipe := utils.NewWebsocketToPipe(ws)
	go wsPipe.Run()
	go ioCopy(cmdPipe, wsPipe, errc)
	go ioCopy(wsPipe, cmdPipe, errc)

dance:
	for {
		select {
		case binaryMessage := <-wsPipe.BinaryChannel():
			dec := gob.NewDecoder(bytes.NewReader(binaryMessage))
			winsize := &utils.Winsize{}
			err = dec.Decode(winsize)
			utils.SetWinsize(cmdPipe.Fd(), winsize)
			if err != nil {
				log.Error("binary-message-error", err)
			} else {
				log.Info("window-resize", lager.Data{"winsize": winsize})
			}
		case err = <-errc:
			if err != nil {
				log.Error("io-error", err)
			}
			err = wsPipe.Close()
			break dance
		}
	}

	// flush all possible messages on errc
flush:
	for {
		select {
		case <-errc:
			if err != nil {
				log.Error("io-error", err)
			}
		default:
			break flush
		}
	}

	log.Info("closing-shell")
	err = cmdPipe.Close()
	if err != nil {
		log.Error("closing-shell-failed", err)
	}

	err = ws.Close()
	if err != nil {
		log.Error("closing-websocket-failed", err)
	}
	log.Info("shell-closed")
}

func ioCopy(dst io.Writer, src io.Reader, errc chan<- error) {
	_, err := io.Copy(dst, src)
	errc <- err
}
