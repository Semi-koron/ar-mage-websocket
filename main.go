package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/Semikoron/armagewebsocket/desktop"
	"github.com/Semikoron/armagewebsocket/mobile"
	"github.com/gorilla/mux"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // ローカル環境用デフォルトポート
	}
	
	r := mux.NewRouter()

	// ルーム作成用の API
	r.HandleFunc("/desktop/{roomID}", desktop.HandleWebSocket)
	r.HandleFunc("/mobile/{roomID}", mobile.HandleWebSocket)

	serverAddr := ":" + port
	fmt.Println("WebSocket server started on", serverAddr)
	log.Fatal(http.ListenAndServe(serverAddr, r))
}