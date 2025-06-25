// internal/websocket/hub.go
package websocket

import (
	"log"
	"sync"
)

type Hub struct {
    // Rooms map[roomID]*Room
    rooms map[string]*Room

    // Stop channel
    stop chan struct{}

    // Mutex for thread-safe operations
    mu sync.RWMutex
}

type Room struct {
    // Registered clients.
    clients map[*Client]bool

    // Inbound messages from the clients.
    Broadcast chan []byte

    // Register requests from the clients.
    Register chan *Client

    // Unregister requests from clients.
    Unregister chan *Client

    // Mutex for thread-safe operations
    mu sync.RWMutex
}

func NewHub() *Hub {
    return &Hub{
        rooms: make(map[string]*Room),
        stop:  make(chan struct{}),
    }
}

func NewRoom() *Room {
    return &Room{
        Broadcast:  make(chan []byte),
        Register:   make(chan *Client),
        Unregister: make(chan *Client),
        clients:    make(map[*Client]bool),
    }
}

func (h *Hub) Run() {
    <-h.stop
    h.mu.Lock()
    for _, room := range h.rooms {
        for client := range room.clients {
            close(client.Send)
        }
    }
    h.mu.Unlock()
}

func (r *Room) Run(hub *Hub, roomID string) {
    for {
        select {
        case client := <-r.Register:
            r.mu.Lock()
            r.clients[client] = true
            r.mu.Unlock()
            
            log.Printf("Client %s connected to room %s. Total clients: %d", client.ID, client.RoomID, len(r.clients))
            
            // 接続通知を他のクライアントに送信
            r.notifyClientJoined(client)

        case client := <-r.Unregister:
            r.mu.Lock()
            if _, ok := r.clients[client]; ok {
                delete(r.clients, client)
                close(client.Send)
            }
            clientCount := len(r.clients)
            r.mu.Unlock()
            
            log.Printf("Client %s disconnected from room %s. Total clients: %d", client.ID, client.RoomID, clientCount)
            
            // 切断通知を他のクライアントに送信
            r.notifyClientLeft(client)
            
            // ルームが空になったらHubから削除
            if clientCount == 0 {
                hub.removeRoom(roomID)
                return
            }

        case message := <-r.Broadcast:
            r.mu.RLock()
            for client := range r.clients {
                select {
                case client.Send <- message:
                default:
                    close(client.Send)
                    delete(r.clients, client)
                }
            }
            r.mu.RUnlock()
        }
    }
}

func (h *Hub) Stop() {
    close(h.stop)
}

func (h *Hub) GetRoom(roomID string) *Room {
    h.mu.Lock()
    defer h.mu.Unlock()
    
    if room, exists := h.rooms[roomID]; exists {
        return room
    }
    
    room := NewRoom()
    h.rooms[roomID] = room
    go room.Run(h, roomID)
    return room
}

func (h *Hub) removeRoom(roomID string) {
    h.mu.Lock()
    defer h.mu.Unlock()
    
    delete(h.rooms, roomID)
    log.Printf("Room %s removed (empty)", roomID)
}

func (h *Hub) GetClientCount() int {
    h.mu.RLock()
    defer h.mu.RUnlock()
    
    total := 0
    for _, room := range h.rooms {
        room.mu.RLock()
        total += len(room.clients)
        room.mu.RUnlock()
    }
    return total
}

func (r *Room) GetClientCount() int {
    r.mu.RLock()
    defer r.mu.RUnlock()
    return len(r.clients)
}

func (r *Room) notifyClientJoined(client *Client) {
    // 実装例
}

func (r *Room) notifyClientLeft(client *Client) {
    // 実装例
}