package market

import (
	"backend-go/model"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ClientConn wraps a websocket connection with a write channel
type ClientConn struct {
	conn     *websocket.Conn
	writeCh  chan map[string]any
	closedMu sync.Mutex
	closed   bool
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
	clientConn := &ClientConn{
		conn:    conn,
		writeCh: make(chan map[string]any, 100), // Buffered channel
		closed:  false,
	}

	b.mu.Lock()
	b.clients[conn] = clientConn
	b.mu.Unlock()

	log.Printf("✅ WebSocket client added. Total clients: %d", len(b.clients))

	// Start a write goroutine for this connection
	go clientConn.writeLoop(b)
}

func (b *WebSocketBroadcaster) RemoveClient(conn *websocket.Conn) {
	b.mu.Lock()
	clientConn, exists := b.clients[conn]
	if exists {
		delete(b.clients, conn)
	}
	b.mu.Unlock()

	if exists {
		clientConn.closedMu.Lock()
		clientConn.closed = true
		clientConn.closedMu.Unlock()
		close(clientConn.writeCh)
		_ = conn.Close()
		log.Printf("❌ WebSocket client removed. Total clients: %d", len(b.clients))
	}
}

// writeLoop runs in its own goroutine and ensures no concurrent writes
func (c *ClientConn) writeLoop(b *WebSocketBroadcaster) {
	for payload := range c.writeCh {
		if err := c.conn.WriteJSON(payload); err != nil {
			log.Println("❌ websocket write error:", err)
			b.RemoveClient(c.conn)
			return
		}
	}
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
		clientConn.closedMu.Lock()
		isClosed := clientConn.closed
		clientConn.closedMu.Unlock()

		if isClosed {
			continue
		}

		// Non-blocking send to write channel
		select {
		case clientConn.writeCh <- payload:
			// Sent successfully
		default:
			// Channel full or closed, skip this message
			log.Println("⚠️ Client channel full, dropping message")
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
