package market

import (
	"backend-go/model"
	"backend-go/store"
	"context"
	"time"
)

// MarketService coordinates market state transitions and engine lifecycle.
type MarketService struct {
	store       *store.Store
	engine      *MarketEngine
	broadcaster *WebSocketBroadcaster
}

func NewMarketService(s *store.Store, broadcaster *WebSocketBroadcaster) *MarketService {
	svc := &MarketService{
		store:       s,
		broadcaster: broadcaster,
	}
	svc.engine = NewMarketEngine(svc, broadcaster, 2*time.Second)
	return svc
}

func (s *MarketService) EnsureEngineRunning() {
	s.engine.Start()
}

func (s *MarketService) StopEngine() {
	s.engine.Stop()
}

func (s *MarketService) StartMarket() (*store.MarketStatus, error) {
	status, err := s.store.StartMarket()
	if err != nil {
		return nil, err
	}
	s.EnsureEngineRunning()
	s.broadcaster.PublishMarketStatus(true)
	return status, nil
}

func (s *MarketService) StopMarket() (*store.MarketStatus, error) {
	status, err := s.store.StopMarket()
	if err != nil {
		return nil, err
	}
	// Hard-stop engine so no simulation cycle can run while market is closed.
	s.StopEngine()
	s.broadcaster.PublishMarketStatus(false)
	return status, nil
}

func (s *MarketService) IsMarketOpen() (bool, error) {
	return s.store.IsMarketOpen()
}

func (s *MarketService) SimulateTickCycle(ctx context.Context, now time.Time) ([]model.StockTick, error) {
	ticks, err := s.store.SimulateTickCycle(ctx, now)
	if err != nil {
		return nil, err
	}
	_ = s.store.CacheLatestTicks(ctx, ticks)
	_ = s.store.PublishRealtimeEvent(ctx, "stock_tick", ticks)
	return ticks, nil
}

func (s *MarketService) GetStocks() ([]model.StockQuote, error) {
	stocks, err := s.store.GetStocks()
	if err != nil {
		return nil, err
	}
	_ = s.store.CacheLatestSnapshot(context.Background(), stocks)
	return stocks, nil
}

func (s *MarketService) GetLatestCandlesForSymbols(symbols []string, timeframe string) ([]model.StockCandle, error) {
	return s.store.GetLatestCandlesForSymbols(symbols, timeframe)
}
