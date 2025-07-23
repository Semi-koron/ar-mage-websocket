// internal/websocket/hub.go
package websocket

import (
	"encoding/json"
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

    // Stage of the room
    stage [][][]int16

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
        stage:     make([][][]int16, 0),
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

func (r *Room) WriteStage(stage [][][]int16) {
    r.mu.Lock()
    r.stage = stage
    r.mu.Unlock()

    log.Printf("Stage updated for room with %d layers", len(stage))
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

// 修正：json.RawMessageに対応
func (r *Room) notifyClientJoined(client *Client) {
    r.mu.RLock()
    stageMsg := StageMessage{Stage: r.stage}
    r.mu.RUnlock()
    
    // StageMessageをjson.RawMessageに変換
    contentBytes, err := json.Marshal(stageMsg)
    if err != nil {
        log.Printf("Error marshaling stage message: %v", err)
        return
    }
    
    msg := Message{
        Type:    "stage",
        Content: json.RawMessage(contentBytes),
        From:    "server",
    }
    
    message, err := json.Marshal(msg)
    if err != nil {
        log.Printf("Error marshaling message: %v", err)
        return
    }
    
    log.Printf("Sending stage data to clients in room. Stage has %d layers", len(stageMsg.Stage))
    
    r.mu.RLock()
    defer r.mu.RUnlock()
    for client := range r.clients {
        select {
        case client.Send <- message:
        default:
            close(client.Send)
            delete(r.clients, client)
        }
    }
}

func (r *Room) notifyClientLeft(client *Client) {
    // 切断通知の実装例
    msg := Message{
        Type:    "client_left",
        Content: json.RawMessage(`{"client_id":"` + client.ID + `"}`),
        From:    "server",
    }
    
    message, err := json.Marshal(msg)
    if err != nil {
        log.Printf("Error marshaling client left message: %v", err)
        return
    }
    
    r.mu.RLock()
    defer r.mu.RUnlock()
    for c := range r.clients {
        if c != client { // 切断したクライアント以外に送信
            select {
            case c.Send <- message:
            default:
                close(c.Send)
                delete(r.clients, c)
            }
        }
    }
}