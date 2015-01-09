package utils

import (
	"time"

	"github.com/gorilla/websocket"
)

type websocketToPipe struct {
	ws            *websocket.Conn
	closing       chan chan error
	binaryChannel chan []byte
	closed        bool
}

func NewWebsocketToPipe(ws *websocket.Conn) *websocketToPipe {
	return &websocketToPipe{
		ws:            ws,
		closing:       make(chan chan error),
		binaryChannel: make(chan []byte),
	}
}

func (wstp *websocketToPipe) BinaryChannel() <-chan []byte {
	return wstp.binaryChannel
}

func (wstp *websocketToPipe) Read(p []byte) (int, error) {
	mType, m, err := wstp.ws.ReadMessage()
	if err != nil {
		return 0, err
	}
	if mType == websocket.TextMessage {
		copy(p, m)
		return len(m), nil
	} else if mType == websocket.BinaryMessage {
		wstp.binaryChannel <- m
		return 0, nil
	}
	return 0, nil
}

func (wstp *websocketToPipe) Write(p []byte) (int, error) {
	err := wstp.ws.WriteMessage(websocket.TextMessage, p)
	return len(p), err
}

func (wstp *websocketToPipe) Run() {
	pingInterval := 5 * time.Second
	wstp.ws.SetPongHandler(func(string) error {
		return wstp.ws.SetReadDeadline(time.Now().Add(pingInterval + 500*time.Millisecond))
	})

	var pong chan error
	for {
		ping := time.After(pingInterval)

		select {
		case <-ping:
			pong = make(chan error)
			go func() {
				pong <- wstp.ws.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(pingInterval/2))
			}()
		case pongerr := <-pong:
			pong = nil
			if pongerr != nil {
				return
			}
		case errc := <-wstp.closing:
			errc <- nil
			return
		}
	}
}

func (wstp *websocketToPipe) Close() error {
	errc := make(chan error)
	wstp.closing <- errc
	return <-errc
}
