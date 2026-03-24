package market

import (
	"backend-go/model"
	"backend-go/store"
	"context"
	"log"
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
	log.Println("🚀 StartMarket called...")
	status, err := s.store.StartMarket()
	if err != nil {
		log.Println("❌ Failed to start market in store:", err)
		return nil, err
	}
	log.Println("✅ Market started in store, starting engine...")
	s.EnsureEngineRunning()
	log.Println("✅ Engine started, publishing status...")
	s.broadcaster.PublishMarketStatus(true)
	log.Println("✅ StartMarket complete")
	return status, nil
}

func (s *MarketService) StopMarket() (*store.MarketStatus, error) {
	log.Println("🛑 StopMarket called...")
	status, err := s.store.StopMarket()
	if err != nil {
		log.Println("❌ Failed to stop market in store:", err)
		return nil, err
	}
	log.Println("✅ Market stopped in store, stopping engine...")
	// Hard-stop engine so no simulation cycle can run while market is closed.
	s.StopEngine()
	log.Println("✅ Engine stopped, publishing status...")
	s.broadcaster.PublishMarketStatus(false)
	log.Println("✅ StopMarket complete")
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
