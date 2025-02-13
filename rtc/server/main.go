package main

import (
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Server struct {
	viewers   map[*websocket.Conn]struct{}
	viewersMu sync.Mutex
}

func main() {
	s := &Server{
		viewers: make(map[*websocket.Conn]struct{}),
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	http.HandleFunc("/ws/viewer", s.handleViewer)
	http.HandleFunc("/ws/client", s.handleClient)

	http.ListenAndServe(":8080", nil)
}

func (s *Server) handleViewer(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	s.viewersMu.Lock()
	s.viewers[conn] = struct{}{}
	s.viewersMu.Unlock()

	for {
		if _, _, err := conn.NextReader(); err != nil {
			break
		}
	}
}

func (s *Server) handleClient(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}
		s.broadcast(message)
	}
}

func (s *Server) broadcast(msg []byte) {
	s.viewersMu.Lock()
	defer s.viewersMu.Unlock()

	for viewer := range s.viewers {
		viewer.WriteMessage(websocket.BinaryMessage, msg)
	}
}
