package api

import (
	"src/logger"
	"sync"

	"github.com/gorilla/websocket"
)

// Hub manages websocket clients and broadcasts messages to them.
type Hub struct {
	mu      sync.RWMutex
	clients map[*websocket.Conn]struct{}
}

func NewHub() *Hub { return &Hub{clients: make(map[*websocket.Conn]struct{})} }

func (h *Hub) Add(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[conn] = struct{}{}
	logger.Info("websocket client connected", logger.FieldKV("remote_addr", conn.RemoteAddr().String()))
}

func (h *Hub) Remove(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, conn)
	_ = conn.Close()
	logger.Info("websocket client disconnected", logger.FieldKV("remote_addr", conn.RemoteAddr().String()))
}

func (h *Hub) Broadcast(msg interface{}) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		if err := c.WriteJSON(msg); err != nil {
			logger.Error("websocket write error", err, logger.FieldKV("remote_addr", c.RemoteAddr().String()))
		}
	}
}

// BroadcastExcept sends the message to all connected clients except the provided connection.
func (h *Hub) BroadcastExcept(msg interface{}, except *websocket.Conn) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		if c == except {
			continue
		}
		if err := c.WriteJSON(msg); err != nil {
			logger.Error("websocket write error", err, logger.FieldKV("remote_addr", c.RemoteAddr().String()))
		}
	}
}
