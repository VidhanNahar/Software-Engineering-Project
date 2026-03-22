package market

import (
	"backend-go/model"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketBroadcaster is a central fan-out hub for real-time events.
type WebSocketBroadcaster struct {
	mu      sync.RWMutex
	clients map[*websocket.Conn]struct{}
}

func NewWebSocketBroadcaster() *WebSocketBroadcaster {
	return &WebSocketBroadcaster{
		clients: make(map[*websocket.Conn]struct{}),
	}
}

func (b *WebSocketBroadcaster) AddClient(conn *websocket.Conn) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.clients[conn] = struct{}{}
}

func (b *WebSocketBroadcaster) RemoveClient(conn *websocket.Conn) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.clients, conn)
}

func (b *WebSocketBroadcaster) broadcast(payload map[string]any) {
	b.mu.RLock()
	clients := make([]*websocket.Conn, 0, len(b.clients))
	for c := range b.clients {
		clients = append(clients, c)
	}
	b.mu.RUnlock()

	for _, conn := range clients {
		if err := conn.WriteJSON(payload); err != nil {
			log.Println("websocket broadcast error:", err)
			b.RemoveClient(conn)
			_ = conn.Close()
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
