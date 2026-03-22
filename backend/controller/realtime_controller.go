package controller

import (
	"backend-go/store"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// RealtimeHandler streams stock snapshots to websocket clients.
type RealtimeHandler struct {
	store *store.Store
}

func NewRealtimeHandler(s *store.Store) *RealtimeHandler {
	return &RealtimeHandler{store: s}
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

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		if err := h.store.RefreshSimulatedPrices(r.Context()); err != nil {
			log.Println("websocket price refresh error:", err)
		}
		if err := h.store.EvaluateAlerts(); err != nil {
			log.Println("websocket alert evaluation error:", err)
		}

		stocks, err := h.store.GetStocks()
		if err != nil {
			_ = conn.WriteJSON(map[string]string{"error": "Failed to fetch stocks"})
			return
		}

		payload := map[string]any{
			"type":      "stocks_snapshot",
			"timestamp": time.Now().UTC(),
			"count":     len(stocks),
			"stocks":    stocks,
		}

		b, _ := json.Marshal(payload)
		if err := conn.WriteMessage(websocket.TextMessage, b); err != nil {
			return
		}

		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
		}
	}
}
