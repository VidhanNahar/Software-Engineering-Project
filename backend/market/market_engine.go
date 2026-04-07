package market

import (
	"backend-go/model"
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// CycleRunner contains the minimum operations the market engine needs per cycle.
type CycleRunner interface {
	IsMarketOpen() (bool, error)
	SimulateTickCycle(ctx context.Context, now time.Time) ([]model.StockTick, error)
	GetStocks() ([]model.StockQuote, error)
	GetLatestCandlesForSymbols(symbols []string, timeframe string) ([]model.StockCandle, error)
}

// MarketEngine is the only component allowed to mutate simulated prices.
type MarketEngine struct {
	runner      CycleRunner
	broadcaster *WebSocketBroadcaster
	interval    time.Duration

	started atomic.Bool
	stopCh  chan struct{}
	wg      sync.WaitGroup
	mu      sync.Mutex
}

func NewMarketEngine(runner CycleRunner, broadcaster *WebSocketBroadcaster, interval time.Duration) *MarketEngine {
	return &MarketEngine{
		runner:      runner,
		broadcaster: broadcaster,
		interval:    interval,
	}
}

func (e *MarketEngine) Start() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.started.Load() {
		return
	}
	e.started.Store(true)

	e.stopCh = make(chan struct{})
	e.wg.Add(1)
	go e.runLoop()
}

func (e *MarketEngine) Stop() {
	e.mu.Lock()
	if !e.started.Load() {
		e.mu.Unlock()
		return
	}
	e.started.Store(false)
	close(e.stopCh)
	e.mu.Unlock()

	// Wait for goroutine to exit gracefully
	e.wg.Wait()
}

func (e *MarketEngine) runLoop() {
	defer e.wg.Done()
	ticker := time.NewTicker(e.interval)
	defer ticker.Stop()

	for {
		select {
		case <-e.stopCh:
			return
		case now := <-ticker.C:
			e.runCycle(now)
		}
	}
}

func (e *MarketEngine) runCycle(now time.Time) {
	ctx, cancel := context.WithTimeout(context.Background(), 18000*time.Millisecond)
	defer cancel()

	isOpen, err := e.runner.IsMarketOpen()
	if err != nil {
		log.Println("❌ market engine status check error:", err)
		return
	}

	if !isOpen {
		return
	}

	log.Println("🔄 Starting market cycle...")
	ticks, err := e.runner.SimulateTickCycle(ctx, now.UTC())
	if err != nil {
		log.Printf("❌ market engine cycle error: %v", err)
		return
	}

	if len(ticks) > 0 {
		log.Printf("✅ Market cycle: generated %d ticks, publishing...", len(ticks))
		e.broadcaster.PublishTickBatch(ticks)
	} else {
		log.Println("⚠️ Market cycle: no ticks generated")
	}

	stocks, err := e.runner.GetStocks()
	if err != nil {
		log.Printf("❌ market engine snapshot error: %v", err)
		return
	}
	if len(stocks) > 0 {
		log.Printf("✅ Publishing snapshot of %d stocks", len(stocks))
		e.broadcaster.PublishSnapshot(stocks, true)
	} else {
		log.Println("⚠️ No stocks available for snapshot")
	}

	symbols := make([]string, 0, len(ticks))
	for _, t := range ticks {
		symbols = append(symbols, t.Symbol)
	}
	candles, err := e.runner.GetLatestCandlesForSymbols(symbols, "1m")
	if err != nil {
		log.Println("❌ market engine candle update error:", err)
		return
	}
	e.broadcaster.PublishCandleUpdates(candles)
}
