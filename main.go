package main

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"unicode/utf8"
	"unsafe"

	"github.com/gorilla/websocket"
	"github.com/kr/pty"
)

var upgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Winsize struct {
	Height uint16
	Width  uint16
	x      uint16
	y      uint16
}

func SetWinsize(fd uintptr, ws *Winsize) error {
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(syscall.TIOCSWINSZ), uintptr(unsafe.Pointer(ws)))
	return err
}

func readLoop(c *websocket.Conn, w *os.File, done chan bool) {
	for {
		mType, m, err := c.ReadMessage()
		if mType == websocket.TextMessage {
			if err != nil {
				log.Println(err)
				done <- true
				return
			}
			w.Write(m)
		} else if mType == websocket.BinaryMessage {
			dec := gob.NewDecoder(bytes.NewReader(m))
			winsize := &Winsize{}
			dec.Decode(winsize)
			SetWinsize(w.Fd(), winsize)
			if err != nil {
				log.Println(err)
				done <- true
				return
			}
		}
	}
}

func writeLoop(c *websocket.Conn, r io.Reader, done chan bool) {
	br := bufio.NewReader(r)
	for {
		x, size, err := br.ReadRune()
		if err != nil {
			log.Println(err)
			done <- true
			return
		}

		p := make([]byte, size)
		utf8.EncodeRune(p, x)

		err = c.WriteMessage(websocket.TextMessage, p)
		if err != nil {
			log.Println(err)
			done <- true
			return
		}
	}
}

func shellHandler(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer ws.Close()

	cmd := exec.Command("/bin/bash", "-l")
	f, err := pty.Start(cmd)

	done := make(chan bool)
	go readLoop(ws, f, done)
	go writeLoop(ws, f, done)
	<-done
}

func main() {
	log.Println("Listening on port 8080")
	http.HandleFunc("/shell", shellHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
