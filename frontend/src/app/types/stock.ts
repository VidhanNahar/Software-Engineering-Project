export interface Stock {
  symbol: string;
  name: string;
  price: number;
  change: number;
  changePercent: number;
  volume: number;
  marketCap: number;
  high: number;
  low: number;
  open: number;
  previousClose: number;
}

export interface WatchlistItem {
  symbol: string;
  name: string;
  price: number;
  change: number;
  changePercent: number;
}

export interface PortfolioHolding {
  symbol: string;
  name: string;
  quantity: number;
  avgPrice: number;
  currentPrice: number;
  totalValue: number;
  totalGainLoss: number;
  gainLossPercent: number;
}

export interface Trade {
  id: string;
  symbol: string;
  type: "buy" | "sell";
  quantity: number;
  price: number;
  total: number;
  timestamp: Date;
  status: "completed" | "pending" | "cancelled";
}

export interface OrderBookEntry {
  price: number;
  quantity: number;
  total: number;
}

export interface ChartDataPoint {
  time: string;
  price: number;
  volume?: number;
}
