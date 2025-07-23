package websocket

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

type Client struct {
	ID     string
	RoomID string
	Hub    *Hub
	Room   *Room
	Conn   *websocket.Conn
	Send   chan []byte
}

// 修正：interface{}をjson.RawMessageに変更
type Message struct {
	Type    string          `json:"type"`
	Content json.RawMessage `json:"content"`
	From    string          `json:"from"`
}

type StageMessage struct {
	Stage [][][]int16 `json:"stage"`
}

func (c *Client) ReadPump() {
	defer func() {
		c.Room.Unregister <- c
		c.Conn.Close()
	}()
	
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	
	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		
		// デバッグ：受信した生メッセージをログ出力
		log.Printf("Received raw message: %s", string(message))
		
		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("error unmarshaling message: %v", err)
			continue
		}
		
		msg.From = c.ID
		
		if msg.Type == "stage" {
			var stageMsg StageMessage
			
			if err := json.Unmarshal(msg.Content, &stageMsg); err != nil {
				log.Printf("error unmarshaling stage message: %v", err)
				log.Printf("content was: %s", string(msg.Content))
				continue
			}
			
			log.Printf("Successfully parsed stage data with %d layers", len(stageMsg.Stage))
			c.Room.WriteStage(stageMsg.Stage)
		}
		
		if processedMessage, err := json.Marshal(msg); err == nil {
			c.Room.Broadcast <- processedMessage
		} else {
			log.Printf("error marshaling processed message: %v", err)
		}
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()
	
	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			
			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)
			
			// 追加のメッセージがある場合の処理
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}
			
			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}