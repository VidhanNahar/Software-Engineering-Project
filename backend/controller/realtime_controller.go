package controller

import (
	"backend-go/market"
	"backend-go/store"
	"net/http"

	"github.com/gorilla/websocket"
)

// RealtimeHandler streams stock snapshots to websocket clients.
type RealtimeHandler struct {
	store       *store.Store
	broadcaster *market.WebSocketBroadcaster
}

func NewRealtimeHandler(s *store.Store, broadcaster *market.WebSocketBroadcaster) *RealtimeHandler {
	return &RealtimeHandler{store: s, broadcaster: broadcaster}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (h *RealtimeHandler) StocksStream(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Failed to upgrade connection", http.StatusBadRequest)
		return
	}
	defer conn.Close()

	h.broadcaster.AddClient(conn)
	defer h.broadcaster.RemoveClient(conn)

	// Send initial snapshot to new subscriber.
	isOpen, _ := h.store.IsMarketOpen()
	stocks, _ := h.store.GetStocks()
	_ = conn.WriteJSON(map[string]any{
		"type":        "stocks_snapshot",
		"count":       len(stocks),
		"market_open": isOpen,
		"stocks":      stocks,
	})

	// Keep the connection alive until client disconnects.
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}
