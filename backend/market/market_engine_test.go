package market

import (
	"backend-go/model"
	"context"
	"errors"
	"testing"
	"time"
)

// --- Mocking the CycleRunner ---

type MockRunner struct {
	IsOpen      bool
	OpenErr     error
	TickErr     error
	Ticks       []model.StockTick
	Stocks      []model.StockQuote
	Candles     []model.StockCandle
	TicksCalled bool
}

func (m *MockRunner) IsMarketOpen() (bool, error) {
	return m.IsOpen, m.OpenErr
}

func (m *MockRunner) SimulateTickCycle(ctx context.Context, now time.Time) ([]model.StockTick, error) {
	m.TicksCalled = true
	return m.Ticks, m.TickErr
}

func (m *MockRunner) GetStocks() ([]model.StockQuote, error) {
	return m.Stocks, nil
}

func (m *MockRunner) GetLatestCandlesForSymbols(symbols []string, timeframe string) ([]model.StockCandle, error) {
	return m.Candles, nil
}

// --- Test Cases ---

// TC-01: Start and Stop Lifecycle
func TestMarketEngine_StartStop(t *testing.T) {
	runner := &MockRunner{IsOpen: true}
	// Note: We use nil for broadcaster here if your real implementation handles nil gracefully,
	// or create a dummy broadcaster. We assume interval is large enough so it doesn't tick immediately.
	engine := NewMarketEngine(runner, nil, 1*time.Hour)

	engine.Start()

	// Wait a tiny bit to let the goroutine spin up
	time.Sleep(10 * time.Millisecond)

	engine.Stop()

	if engine.started.Load() {
		t.Error("Expected engine to be stopped, but started=true")
	}
}

// TC-02: Idempotent Start
func TestMarketEngine_MultipleStarts(t *testing.T) {
	runner := &MockRunner{IsOpen: true}
	engine := NewMarketEngine(runner, nil, 1*time.Hour)

	// Call Start twice
	engine.Start()
	engine.Start()

	time.Sleep(10 * time.Millisecond)
	engine.Stop()
	// If it doesn't panic or deadlock, the test passes.
}

// TC-03: Market Closed
func TestMarketEngine_MarketClosed(t *testing.T) {
	runner := &MockRunner{IsOpen: false}
	engine := NewMarketEngine(runner, nil, 1*time.Hour)

	// Trigger a single cycle manually instead of waiting for the ticker
	engine.runCycle(time.Now())

	if runner.TicksCalled {
		t.Error("Expected SimulateTickCycle NOT to be called when market is closed")
	}
}

// TC-05: Error in IsMarketOpen
func TestMarketEngine_ErrorInStatus(t *testing.T) {
	runner := &MockRunner{
		IsOpen:  false,
		OpenErr: errors.New("database connection failed"),
	}
	engine := NewMarketEngine(runner, nil, 1*time.Hour)

	engine.runCycle(time.Now())

	if runner.TicksCalled {
		t.Error("Expected SimulateTickCycle NOT to be called when IsMarketOpen returns an error")
	}
}
