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
	clients map[*ClientConn]struct{}
}

func NewWebSocketBroadcaster() *WebSocketBroadcaster {
	return &WebSocketBroadcaster{
		clients: make(map[*ClientConn]struct{}),
	}
}

func (b *WebSocketBroadcaster) AddClient(conn *websocket.Conn) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.clients[&ClientConn{conn: conn}] = struct{}{}
}

func (b *WebSocketBroadcaster) RemoveClient(conn *websocket.Conn) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for client := range b.clients {
		if client.conn == conn {
			delete(b.clients, client)
			break
		}
	}
}

func (b *WebSocketBroadcaster) broadcast(payload map[string]any) {
	b.mu.RLock()
	clients := make([]*ClientConn, 0, len(b.clients))
	for c := range b.clients {
		clients = append(clients, c)
	}
	b.mu.RUnlock()

	for _, clientConn := range clients {
		clientConn.mu.Lock()
		err := clientConn.conn.WriteJSON(payload)
		clientConn.mu.Unlock()

		if err != nil {
			log.Println("websocket broadcast error:", err)
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
		return
	}
	b.broadcast(map[string]any{
		"type":      "stock_tick",
		"timestamp": time.Now().UTC(),
		"count":     len(ticks),
		"ticks":     ticks,
	})
}

func (b *WebSocketBroadcaster) PublishSnapshot(stocks []model.StockQuote, isOpen bool) {
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
