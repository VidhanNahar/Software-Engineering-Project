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
	doneCh  chan struct{}
	mu      sync.Mutex
}

func NewMarketEngine(runner CycleRunner, broadcaster *WebSocketBroadcaster, interval time.Duration) *MarketEngine {
	return &MarketEngine{
		runner:      runner,
		broadcaster: broadcaster,
		interval:    interval,
		stopCh:      make(chan struct{}),
		doneCh:      make(chan struct{}),
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
	e.doneCh = make(chan struct{})
	go e.runLoop()
}

func (e *MarketEngine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.started.Load() {
		return
	}
	close(e.stopCh)
	e.mu.Unlock()

	// Wait for goroutine to exit gracefully
	<-e.doneCh

	e.mu.Lock()
	e.started.Store(false)
}

func (e *MarketEngine) runLoop() {
	defer close(e.doneCh)
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
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()

	isOpen, err := e.runner.IsMarketOpen()
	if err != nil {
		log.Println("market engine status check error:", err)
		return
	}

	e.broadcaster.PublishMarketStatus(isOpen)
	if !isOpen {
		return
	}

	ticks, err := e.runner.SimulateTickCycle(ctx, now.UTC())
	if err != nil {
		log.Println("market engine cycle error:", err)
		return
	}

	e.broadcaster.PublishTickBatch(ticks)

	stocks, err := e.runner.GetStocks()
	if err != nil {
		log.Println("market engine snapshot error:", err)
		return
	}
	e.broadcaster.PublishSnapshot(stocks, true)

	symbols := make([]string, 0, len(ticks))
	for _, t := range ticks {
		symbols = append(symbols, t.Symbol)
	}
	candles, err := e.runner.GetLatestCandlesForSymbols(symbols, "1m")
	if err != nil {
		log.Println("market engine candle update error:", err)
		return
	}
	e.broadcaster.PublishCandleUpdates(candles)
}
