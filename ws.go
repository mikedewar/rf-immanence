package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

type connection struct {
	ws   *websocket.Conn
	send chan []byte
}

func (c *connection) write(mt int, payload []byte) error {
	c.ws.SetWriteDeadline(time.Now().Add(writeWait))
	return c.ws.WriteMessage(mt, payload)
}

type Server struct {
	addConn   chan *connection
	delConn   chan *connection
	broadcast chan []byte
	emitChan  chan []byte
}

func (s *Server) websocketRouter() {
	hub := make(map[*connection]bool)
	for {
		select {
		case c := <-s.addConn:
			hub[c] = true
		case c := <-s.delConn:
			log.Println("goodbye")
			delete(hub, c)
		case m := <-s.broadcast:
			for c, _ := range hub {
				c.send <- m
			}
		}
	}
}

func (s *Server) websocketReadPump(c *connection) {
	defer func() {
		s.delConn <- c
		c.ws.Close()
	}()
	c.ws.SetReadLimit(maxMessageSize)
	c.ws.SetReadDeadline(time.Now().Add(pongWait))
	c.ws.SetPongHandler(func(string) error { c.ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.ws.ReadMessage()
		if err != nil {
			break
		}
		s.emitChan <- message
	}
}

func (s *Server) websocketWritePump(c *connection) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.ws.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.write(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.write(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.write(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

func (s *Server) websocketHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	c := &connection{send: make(chan []byte, 256), ws: ws}

	s.addConn <- c
	go s.websocketWritePump(c)
	go s.websocketReadPump(c)
}

func (s *Server) addHandler(w http.ResponseWriter, r *http.Request) {

	s.broadcast <- []byte(fmt.Sprintf(`{"server":"hi nik"}`))

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"server":"ok"}`))
}
