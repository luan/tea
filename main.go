package main

import (
	"bufio"
	"io"
	"log"
	"net/http"
	"os/exec"
	"unicode/utf8"

	"github.com/gorilla/websocket"
	"github.com/kr/pty"
)

var upgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func readLoop(c *websocket.Conn, w io.Writer) {
	for {
		_, m, err := c.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}
		w.Write(m)
	}
}

func writeLoop(c *websocket.Conn, r io.Reader) {
	br := bufio.NewReader(r)
	for {
		x, size, _ := br.ReadRune()
		p := make([]byte, size)
		utf8.EncodeRune(p, x)

		err := c.WriteMessage(websocket.TextMessage, p)
		if err != nil {
			panic(err)
		}
	}
}

func shellHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	cmd := exec.Command("/bin/bash", "-l")
	f, err := pty.Start(cmd)

	go readLoop(conn, f)
	writeLoop(conn, f)
}

func main() {
	http.HandleFunc("/shell", shellHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
