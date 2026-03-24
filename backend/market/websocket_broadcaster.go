package market

import (
	"backend-go/model"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ClientConn wraps a websocket connection with its own write mutex
type ClientConn struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

// WebSocketBroadcaster is a central fan-out hub for real-time events.
type WebSocketBroadcaster struct {
	mu      sync.RWMutex
	clients map[*websocket.Conn]*ClientConn
}

func NewWebSocketBroadcaster() *WebSocketBroadcaster {
	return &WebSocketBroadcaster{
		clients: make(map[*websocket.Conn]*ClientConn),
	}
}

func (b *WebSocketBroadcaster) AddClient(conn *websocket.Conn) {
	b.mu.Lock()
	defer b.mu.Unlock()
	// Only add if not already present
	if _, exists := b.clients[conn]; !exists {
		b.clients[conn] = &ClientConn{conn: conn}
		log.Printf("✅ WebSocket client added. Total clients: %d", len(b.clients))
	}
}

func (b *WebSocketBroadcaster) RemoveClient(conn *websocket.Conn) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.clients, conn)
	log.Printf("❌ WebSocket client removed. Total clients: %d", len(b.clients))
}

func (b *WebSocketBroadcaster) broadcast(payload map[string]any) {
	b.mu.RLock()
	clients := make([]*ClientConn, 0, len(b.clients))
	for _, c := range b.clients {
		clients = append(clients, c)
	}
	b.mu.RUnlock()

	if len(clients) == 0 {
		log.Printf("⚠️ No WebSocket clients connected, message not sent: type=%v", payload["type"])
		return
	}

	log.Printf("📤 Broadcasting to %d clients: type=%v", len(clients), payload["type"])

	for _, clientConn := range clients {
		clientConn.mu.Lock()
		err := clientConn.conn.WriteJSON(payload)
		clientConn.mu.Unlock()

		if err != nil {
			log.Println("❌ websocket broadcast error:", err)
			b.RemoveClient(clientConn.conn)
			_ = clientConn.conn.Close()
		}
	}
}

func (b *WebSocketBroadcaster) PublishMarketStatus(isOpen bool) {
	b.broadcast(map[string]any{
		"type":        "market_status",
		"timestamp":   time.Now().UTC(),
		"market_open": isOpen,
	})
}

func (b *WebSocketBroadcaster) PublishTickBatch(ticks []model.StockTick) {
	if len(ticks) == 0 {
		log.Println("⚠️ PublishTickBatch called with 0 ticks")
		return
	}
	log.Printf("📊 PublishTickBatch: %d ticks", len(ticks))
	b.broadcast(map[string]any{
		"type":      "stock_tick",
		"timestamp": time.Now().UTC(),
		"count":     len(ticks),
		"ticks":     ticks,
	})
}

func (b *WebSocketBroadcaster) PublishSnapshot(stocks []model.StockQuote, isOpen bool) {
	log.Printf("📸 PublishSnapshot: %d stocks, market_open=%v", len(stocks), isOpen)
	b.broadcast(map[string]any{
		"type":        "stocks_snapshot",
		"timestamp":   time.Now().UTC(),
		"count":       len(stocks),
		"market_open": isOpen,
		"stocks":      stocks,
	})
}

func (b *WebSocketBroadcaster) PublishCandleUpdates(candles []model.StockCandle) {
	if len(candles) == 0 {
		return
	}
	b.broadcast(map[string]any{
		"type":      "candle_update",
		"timestamp": time.Now().UTC(),
		"count":     len(candles),
		"candles":   candles,
	})
}
