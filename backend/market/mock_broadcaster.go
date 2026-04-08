package market

import (
	"backend-go/model"
	"backend-go/store"
	"context"
	"log"
	"math"
	"math/rand"
	"sync/atomic"
	"time"
)

// MockPriceBroadcaster generates and broadcasts simulated prices without database hits
type MockPriceBroadcaster struct {
	broadcaster *WebSocketBroadcaster
	store       *store.Store
	stocks      []model.StockQuote
	stopCh      chan struct{}
	matching    atomic.Bool
}

func NewMockPriceBroadcaster(broadcaster *WebSocketBroadcaster, s *store.Store) *MockPriceBroadcaster {
	return &MockPriceBroadcaster{
		broadcaster: broadcaster,
		store:       s,
		stocks:      []model.StockQuote{},
		stopCh:      make(chan struct{}),
	}
}

func (m *MockPriceBroadcaster) SetStocks(stocks []model.StockQuote) {
	m.stocks = stocks
}

func (m *MockPriceBroadcaster) Start() {
	go m.run()
}

func (m *MockPriceBroadcaster) Stop() {
	close(m.stopCh)
}

func (m *MockPriceBroadcaster) run() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for {
		select {
		case <-m.stopCh:
			log.Println("🛑 Mock price broadcaster stopped")
			return
		case <-ticker.C:
			// Safely check if we have stocks to broadcast
			if len(m.stocks) > 0 {
				m.broadcastMockPrices(rng)
			}
		}
	}
}

func (m *MockPriceBroadcaster) broadcastMockPrices(rng *rand.Rand) {
	if len(m.stocks) == 0 {
		return
	}

	ticks := make([]model.StockTick, 0, len(m.stocks))
	ticksByStockID := make(map[string]float64, len(m.stocks))

	for i := range m.stocks {
		stock := &m.stocks[i]
		// Generate realistic price movement
		volatility := 0.02 // 2% volatility
		drift := 0.0001    // slight upward drift
		shock := rng.NormFloat64() * volatility

		logReturn := drift + shock
		nextPrice := stock.Price * math.Exp(logReturn)

		// Clamp price to reasonable range
		if nextPrice < 0.5 {
			nextPrice = 0.5
		}
		if nextPrice > stock.Price*1.05 {
			nextPrice = stock.Price * 1.05
		}
		if nextPrice < stock.Price*0.95 {
			nextPrice = stock.Price * 0.95
		}

		// Round to 2 decimals
		nextPrice = math.Round(nextPrice*100) / 100

		// Update stock price
		stock.Price = nextPrice

		// Create tick record
		qty := int64(rng.Intn(50) + 1)
		ticks = append(ticks, model.StockTick{
			Symbol:     stock.Symbol,
			TickTime:   time.Now().UTC(),
			Price:      nextPrice,
			Volume:     qty,
			TradeValue: nextPrice * float64(qty),
		})
		ticksByStockID[stock.StockID.String()] = nextPrice
	}

	// Broadcast to WebSocket clients
	if len(ticks) > 0 {
		// Check market status
		marketStatus, err := m.store.GetMarketStatus()
		isOpen := true
		if err == nil && marketStatus != nil {
			isOpen = marketStatus.IsOpen
		}

		// Broadcast with market status included
		m.broadcaster.broadcast(map[string]any{
			"type":        "stock_tick",
			"timestamp":   time.Now().UTC(),
			"count":       len(ticks),
			"ticks":       ticks,
			"market_open": isOpen,
		})

		// Run pending-order matching asynchronously so websocket tick cadence stays at 2 seconds.
		if isOpen {
			m.matchPendingOrdersAsync(ticksByStockID)
		}
	}
}

func (m *MockPriceBroadcaster) matchPendingOrdersAsync(ticksByStockID map[string]float64) {
	if !m.matching.CompareAndSwap(false, true) {
		return
	}

	go func() {
		defer m.matching.Store(false)

		for i := range m.stocks {
			stock := m.stocks[i]
			price, ok := ticksByStockID[stock.StockID.String()]
			if !ok {
				continue
			}

			ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
			err := m.store.MatchPendingOrders(ctx, stock.StockID, price)
			cancel()
			if err != nil {
				log.Printf("⚠️ Error matching pending orders for stock %s: %v", stock.Symbol, err)
			}
		}
	}()
}
