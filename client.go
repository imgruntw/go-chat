package main

import (
	"bytes"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"time"
)

const (
	// time allowed to write a message to the peer
	writeWait = 10 * time.Second
	// time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second
	// send pings to peer with this period, must be less than pongWait
	pingPeriod = (pongWait * 9) / 10
	// maximum message size allowed from peer
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// client is a middleman between the websocket connection and the hub
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	// buffered channel of outbound messages
	send chan []byte
}

// pumps messages from the websocket connection to the hub
func (client *Client) readPump() {
	defer func() {
		client.hub.unregister <- client
		if err := client.conn.Close(); err != nil {
			log.Printf("error: %v", err)
		}
	}()

	client.conn.SetReadLimit(maxMessageSize)
	if err := client.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.Printf("error: %v", err)
	}
	client.conn.SetPongHandler(func(string) error {
		if err := client.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
			log.Printf("error: %v", err)
		}
		return nil
	})

	for {
		_, message, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		client.hub.broadcast <- message
	}
}

// pumps messages from the hub to the websocket connection
func (client *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		if err := client.conn.Close(); err != nil {
			log.Printf("error: %v", err)
		}
	}()

	for {
		select {
		case message, ok := <-client.send:
			if err := client.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				log.Printf("error: %v", err)
			}

			if !ok {
				// hub closed the channel
				if err := client.conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					log.Printf("error: %v", err)
				}

				return
			}

			w, err := client.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				log.Printf("error: %v", err)
				return
			}

			if _, err := w.Write(message); err != nil {
				log.Printf("error: %v", err)
			}

			// add queued chat messages to the current websocket message
			n := len(client.send)
			for i := 0; i < n; i++ {
				if _, err := w.Write(newline); err != nil {
					log.Printf("error: %v", err)
				}

				if _, err := w.Write(<-client.send); err != nil {
					log.Printf("error: %v", err)
				}
			}

			if err := w.Close(); err != nil {
				log.Printf("error: %v", err)
				return
			}
		case <-ticker.C:
			if err := client.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				log.Printf("error: %v", err)
			}

			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("error: %v", err)
				return
			}
		}
	}
}

// handles websocket requests from the peer
func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}

	client := &Client{hub: hub, conn: conn, send: make(chan []byte, 256)}
	client.hub.register <- client

	// allow collection of memory referenced by the caller by doing all work in new goroutines
	go client.writePump()
	go client.readPump()
}
