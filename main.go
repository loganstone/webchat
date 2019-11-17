package main

import (
	"flag"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/loganstone/trace"
)

const (
	socketBufferSize  = 1024
	messageBufferSize = 256
)

var upgrader = &websocket.Upgrader{
	ReadBufferSize:  socketBufferSize,
	WriteBufferSize: socketBufferSize,
}
var host = flag.String("host", ":8080", "The host of the application.")

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

type room struct {
	forward chan []byte
	join    chan *client
	leave   chan *client
	clients map[*client]bool
	tracer  trace.Tracer
}

func newRoom() *room {
	return &room{
		forward: make(chan []byte),
		join:    make(chan *client),
		leave:   make(chan *client),
		clients: make(map[*client]bool),
		tracer:  trace.Off(),
	}
}

func (r *room) run() {
	for {
		select {
		case client := <-r.join:
			r.clients[client] = true
			r.tracer.Trace("New client join")
		case client := <-r.leave:
			delete(r.clients, client)
			close(client.send)
			r.tracer.Trace("Client left")
		case message := <-r.forward:
			r.tracer.Trace("Message received: ", string(message))
			for client := range r.clients {
				client.send <- message
				r.tracer.Trace(" -- sent to client")
			}
		}
	}
}

func (r *room) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	socket, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Fatal("Server Fatal:", err)
	}
	client := &client{
		socket: socket,
		send:   make(chan []byte, messageBufferSize),
		room:   r,
	}
	r.join <- client
	defer func() {
		r.leave <- client
	}()
	go client.write()
	client.read()
}

type templateHandler struct {
	once     sync.Once
	filename string
	templ    *template.Template
}

func (t *templateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	t.once.Do(func() {
		filesPath := filepath.Join("templates", t.filename)
		t.templ = template.Must(template.ParseFiles(filesPath))
	})
	t.templ.Execute(w, r)
}

func test(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Test"))
}

func init() {
	flag.Parse()
}

func main() {
	fs := http.FileServer(http.Dir("node_modules"))
	r := newRoom()
	r.tracer = trace.New(os.Stdout)

	http.Handle("/static/", http.StripPrefix("/static/", fs))
	http.Handle("/", &templateHandler{filename: "chat.html"})
	http.Handle("/room", r)

	go r.run()

	log.Println("Starting web server on", *host)
	if err := http.ListenAndServe(*host, nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
