package handler

import (
	"log"
	"net/http"
	"time"

	"github.com/Semikoron/ar-mage-websocket/internal/websocket"
	"github.com/gorilla/mux"
	ws "github.com/gorilla/websocket"
)

var upgrader = ws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func HandleWebSocket(hub *websocket.Hub) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        conn, err := upgrader.Upgrade(w, r, nil)
        if err != nil {
            log.Printf("WebSocket upgrade failed: %v", err)
            return
        }

        // Get roomID from path parameters or query string
        vars := mux.Vars(r)
        roomID := vars["roomID"]
        if roomID == "" {
            roomID = r.URL.Query().Get("room")
        }
        if roomID == "" {
            roomID = "default"
        }

        // Get room for this roomID
        room := hub.GetRoom(roomID)

        clientID := generateClientID()
        client := &websocket.Client{
            ID:     clientID,
            RoomID: roomID,
            Hub:    hub,
            Room:   room,
            Conn:   conn,
            Send:   make(chan []byte, 256),
        }

        room.Register <- client

        go client.WritePump()
        go client.ReadPump()
    }
}

func generateClientID() string {
    return time.Now().Format("20060102150405") + "-" + randString(6)
}

func randString(n int) string {
    const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
    b := make([]byte, n)
    for i := range b {
        b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
    }
    return string(b)
}