package main

import (
	"flag"
	"log"
	"net/http"
)

var addr = flag.String("addr", ":8888", "http service address")

func serveChatPage(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)

	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	http.ServeFile(w, r, "go-chat.html")
}

func main() {
	flag.Parse()

	log.Printf("starting websocket server on port: %v", *addr)

	hub := newHub()
	go hub.run()

	http.HandleFunc("/", serveChatPage)
	http.HandleFunc("/ws", func(resWriter http.ResponseWriter, req *http.Request) {
		serveWs(hub, resWriter, req)
	})

	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal()
	}
}
