// internal/websocket/hub.go
package websocket

import (
	"log"
	"sync"
)

type Hub struct {
    // Registered clients.
    clients map[*Client]bool

    // Inbound messages from the clients.
    Broadcast chan []byte

    // Register requests from the clients.
    Register chan *Client

    // Unregister requests from clients.
    Unregister chan *Client

    // Stop channel
    stop chan struct{}

    // Mutex for thread-safe operations
    mu sync.RWMutex
}

func NewHub() *Hub {
    return &Hub{
        Broadcast:  make(chan []byte),
        Register:   make(chan *Client),
        Unregister: make(chan *Client),
        clients:    make(map[*Client]bool),
        stop:       make(chan struct{}),
    }
}

func (h *Hub) Run() {
    for {
        select {
        case client := <-h.Register:
            h.mu.Lock()
            h.clients[client] = true
            h.mu.Unlock()
            
            log.Printf("Client %s connected. Total clients: %d", client.ID, len(h.clients))
            
            // 接続通知を他のクライアントに送信
            h.notifyClientJoined(client)

        case client := <-h.Unregister:
            h.mu.Lock()
            if _, ok := h.clients[client]; ok {
                delete(h.clients, client)
                close(client.Send)
            }
            h.mu.Unlock()
            
            log.Printf("Client %s disconnected. Total clients: %d", client.ID, len(h.clients))
            
            // 切断通知を他のクライアントに送信
            h.notifyClientLeft(client)

        case message := <-h.Broadcast:
            h.mu.RLock()
            for client := range h.clients {
                select {
                case client.Send <- message:
                default:
                    close(client.Send)
                    delete(h.clients, client)
                }
            }
            h.mu.RUnlock()

        case <-h.stop:
            h.mu.Lock()
            for client := range h.clients {
                close(client.Send)
            }
            h.mu.Unlock()
            return
        }
    }
}

func (h *Hub) Stop() {
    close(h.stop)
}

func (h *Hub) GetClientCount() int {
    h.mu.RLock()
    defer h.mu.RUnlock()
    return len(h.clients)
}

func (h *Hub) notifyClientJoined(client *Client) {
    // 実装例
}

func (h *Hub) notifyClientLeft(client *Client) {
    // 実装例
}