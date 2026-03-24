export type Timeframe = "1D" | "5D" | "1M" | "6M" | "1Y";

export interface CandlePoint {
  time: number; // unix seconds
  open: number;
  high: number;
  low: number;
  close: number;
  volume: number;
}

export interface StockOption {
  stockId: string;
  symbol: string;
  name: string;
}

export interface LiveFeedState {
  candles: CandlePoint[];
  price: number;
  connected: boolean;
  lastUpdatedAt?: number;
  marketOpen: boolean;
}
