import { Stock, WatchlistItem, PortfolioHolding, Trade, ChartDataPoint } from "../types/stock";

// Mock stock data
export const mockStocks: Record<string, Stock> = {
  AAPL: {
    symbol: "AAPL",
    name: "Apple Inc.",
    price: 178.45,
    change: 2.34,
    changePercent: 1.33,
    volume: 58234567,
    marketCap: 2800000000000,
    high: 179.23,
    low: 176.12,
    open: 177.00,
    previousClose: 176.11,
  },
  GOOGL: {
    symbol: "GOOGL",
    name: "Alphabet Inc.",
    price: 142.67,
    change: -1.23,
    changePercent: -0.85,
    volume: 23456789,
    marketCap: 1800000000000,
    high: 144.56,
    low: 142.34,
    open: 143.90,
    previousClose: 143.90,
  },
  MSFT: {
    symbol: "MSFT",
    name: "Microsoft Corporation",
    price: 412.34,
    change: 5.67,
    changePercent: 1.39,
    volume: 34567890,
    marketCap: 3100000000000,
    high: 414.23,
    low: 409.45,
    open: 410.00,
    previousClose: 406.67,
  },
  TSLA: {
    symbol: "TSLA",
    name: "Tesla, Inc.",
    price: 189.23,
    change: -3.45,
    changePercent: -1.79,
    volume: 89234567,
    marketCap: 600000000000,
    high: 192.34,
    low: 188.90,
    open: 191.50,
    previousClose: 192.68,
  },
  AMZN: {
    symbol: "AMZN",
    name: "Amazon.com Inc.",
    price: 178.92,
    change: 1.87,
    changePercent: 1.06,
    volume: 45678901,
    marketCap: 1850000000000,
    high: 179.89,
    low: 177.23,
    open: 178.00,
    previousClose: 177.05,
  },
  NVDA: {
    symbol: "NVDA",
    name: "NVIDIA Corporation",
    price: 878.45,
    change: 12.34,
    changePercent: 1.42,
    volume: 67890123,
    marketCap: 2200000000000,
    high: 882.67,
    low: 871.23,
    open: 874.00,
    previousClose: 866.11,
  },
  META: {
    symbol: "META",
    name: "Meta Platforms Inc.",
    price: 489.56,
    change: 7.89,
    changePercent: 1.64,
    volume: 34567890,
    marketCap: 1250000000000,
    high: 492.34,
    low: 485.67,
    open: 487.00,
    previousClose: 481.67,
  },
  NFLX: {
    symbol: "NFLX",
    name: "Netflix Inc.",
    price: 634.23,
    change: -2.45,
    changePercent: -0.38,
    volume: 12345678,
    marketCap: 280000000000,
    high: 638.90,
    low: 632.45,
    open: 636.00,
    previousClose: 636.68,
  },
};

export const defaultWatchlist: WatchlistItem[] = [
  {
    symbol: "AAPL",
    name: "Apple Inc.",
    price: 178.45,
    change: 2.34,
    changePercent: 1.33,
  },
  {
    symbol: "GOOGL",
    name: "Alphabet Inc.",
    price: 142.67,
    change: -1.23,
    changePercent: -0.85,
  },
  {
    symbol: "MSFT",
    name: "Microsoft Corporation",
    price: 412.34,
    change: 5.67,
    changePercent: 1.39,
  },
  {
    symbol: "TSLA",
    name: "Tesla, Inc.",
    price: 189.23,
    change: -3.45,
    changePercent: -1.79,
  },
  {
    symbol: "NVDA",
    name: "NVIDIA Corporation",
    price: 878.45,
    change: 12.34,
    changePercent: 1.42,
  },
];

export const defaultPortfolio: PortfolioHolding[] = [
  {
    symbol: "AAPL",
    name: "Apple Inc.",
    quantity: 50,
    avgPrice: 165.00,
    currentPrice: 178.45,
    totalValue: 8922.50,
    totalGainLoss: 672.50,
    gainLossPercent: 8.15,
  },
  {
    symbol: "MSFT",
    name: "Microsoft Corporation",
    quantity: 25,
    avgPrice: 380.00,
    currentPrice: 412.34,
    totalValue: 10308.50,
    totalGainLoss: 808.50,
    gainLossPercent: 8.51,
  },
  {
    symbol: "NVDA",
    name: "NVIDIA Corporation",
    quantity: 15,
    avgPrice: 720.00,
    currentPrice: 878.45,
    totalValue: 13176.75,
    totalGainLoss: 2376.75,
    gainLossPercent: 22.01,
  },
];

export const recentTrades: Trade[] = [
  {
    id: "1",
    symbol: "AAPL",
    type: "buy",
    quantity: 10,
    price: 178.45,
    total: 1784.50,
    timestamp: new Date(Date.now() - 3600000),
    status: "completed",
  },
  {
    id: "2",
    symbol: "GOOGL",
    type: "sell",
    quantity: 5,
    price: 142.67,
    total: 713.35,
    timestamp: new Date(Date.now() - 7200000),
    status: "completed",
  },
  {
    id: "3",
    symbol: "NVDA",
    type: "buy",
    quantity: 3,
    price: 878.45,
    total: 2635.35,
    timestamp: new Date(Date.now() - 10800000),
    status: "completed",
  },
];

// Generate chart data for stock
export const generateChartData = (basePrice: number, days: number = 30): ChartDataPoint[] => {
  const data: ChartDataPoint[] = [];
  let price = basePrice * 0.95; // Start slightly lower
  
  for (let i = days; i >= 0; i--) {
    const date = new Date();
    date.setDate(date.getDate() - i);
    
    // Add some randomness to simulate price movement
    const change = (Math.random() - 0.5) * basePrice * 0.02;
    price += change;
    
    data.push({
      time: date.toLocaleDateString("en-US", { month: "short", day: "numeric" }),
      price: parseFloat(price.toFixed(2)),
      volume: Math.floor(Math.random() * 10000000) + 5000000,
    });
  }
  
  return data;
};

// Generate intraday chart data
export const generateIntradayData = (basePrice: number): ChartDataPoint[] => {
  const data: ChartDataPoint[] = [];
  let price = basePrice * 0.99;
  
  const now = new Date();
  const marketOpen = new Date(now);
  marketOpen.setHours(9, 30, 0, 0);
  
  for (let i = 0; i < 390; i += 5) { // Every 5 minutes for trading day
    const time = new Date(marketOpen.getTime() + i * 60000);
    const change = (Math.random() - 0.5) * basePrice * 0.003;
    price += change;
    
    data.push({
      time: time.toLocaleTimeString("en-US", { hour: "numeric", minute: "2-digit" }),
      price: parseFloat(price.toFixed(2)),
      volume: Math.floor(Math.random() * 100000) + 50000,
    });
  }
  
  return data;
};

// Simulate real-time price updates
export const simulatePriceUpdate = (currentPrice: number): number => {
  const change = (Math.random() - 0.5) * currentPrice * 0.001;
  return parseFloat((currentPrice + change).toFixed(2));
};
