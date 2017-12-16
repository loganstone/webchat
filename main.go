package main

import (
	"flag"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"sync"
)

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

func main() {
	var addr = flag.String("addr", ":8080", "The addr of the application.")
	flag.Parse()

	fs := http.FileServer(http.Dir("node_modules"))
	r := newRoom()

	http.Handle("/static/", http.StripPrefix("/static/", fs))
	http.Handle("/", &templateHandler{filename: "chat.html"})
	http.Handle("/room", r)

	go r.run()

	log.Println("Starting web server on", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
