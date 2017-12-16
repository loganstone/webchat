package main

import "github.com/gorilla/websocket"

type client struct {
	socket *websocket.Conn
	send   chan []byte
	room   *room
}

func (c *client) read() {
	defer c.socket.Close()
	for {
		_, message, err := c.socket.ReadMessage()
		if err != nil {
			return
		}
		c.room.forward <- message
	}
}

func (c *client) write() {
	defer c.socket.Close()
	for message := range c.send {
		err := c.socket.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			return
		}
	}
}
