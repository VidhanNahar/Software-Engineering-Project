package store

import (
	"backend-go/model"
	"context"
	"encoding/json"
	"fmt"
	"time"
)

func (s *Store) CacheLatestSnapshot(ctx context.Context, stocks []model.StockQuote) error {
	payload, err := json.Marshal(stocks)
	if err != nil {
		return err
	}
	return s.rdb.Set(ctx, "finxgrow:stocks:snapshot", payload, 10*time.Second).Err()
}

func (s *Store) CacheLatestTicks(ctx context.Context, ticks []model.StockTick) error {
	if len(ticks) == 0 {
		return nil
	}

	pipe := s.rdb.Pipeline()
	for _, tick := range ticks {
		payload, err := json.Marshal(tick)
		if err != nil {
			return err
		}
		key := fmt.Sprintf("finxgrow:stock:latest_tick:%s", tick.Symbol)
		pipe.Set(ctx, key, payload, 30*time.Second)
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (s *Store) PublishRealtimeEvent(ctx context.Context, eventType string, payload any) error {
	body, err := json.Marshal(map[string]any{
		"type":      eventType,
		"timestamp": time.Now().UTC(),
		"payload":   payload,
	})
	if err != nil {
		return err
	}
	return s.rdb.Publish(ctx, "finxgrow:market:events", body).Err()
}
