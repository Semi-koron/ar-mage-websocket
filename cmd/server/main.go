package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/Semikoron/ar-mage-websocket/internal/handler"
	"github.com/Semikoron/ar-mage-websocket/internal/websocket"
	"github.com/gorilla/mux"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	hub := websocket.NewHub()
	go hub.Run()
	
	r := mux.NewRouter()

	r.HandleFunc("/ws", handler.HandleWebSocket(hub))
	r.HandleFunc("/desktop/{roomID}", handler.HandleWebSocket(hub))
	r.HandleFunc("/mobile/{roomID}", handler.HandleWebSocket(hub))

	serverAddr := ":" + port
	fmt.Println("WebSocket server started on", serverAddr)
	log.Fatal(http.ListenAndServe(serverAddr, r))
}