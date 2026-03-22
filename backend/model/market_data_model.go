package model

import "time"

// StockTick is one simulated market tick per symbol.
type StockTick struct {
	Symbol    string    `json:"symbol"`
	TickTime  time.Time `json:"tick_time"`
	Price     float64   `json:"price"`
	Volume    int64     `json:"volume"`
	TradeValue float64  `json:"trade_value"`
}

// StockCandle is OHLCV data for a bucket interval.
type StockCandle struct {
	Symbol     string    `json:"symbol"`
	Timeframe  string    `json:"timeframe"`
	CandleTime time.Time `json:"candle_time"`
	Open       float64   `json:"open"`
	High       float64   `json:"high"`
	Low        float64   `json:"low"`
	Close      float64   `json:"close"`
	Volume     int64     `json:"volume"`
	TickCount  int64     `json:"tick_count"`
}
