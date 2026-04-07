import { useEffect, useMemo, useState, useRef } from "react";
import { useNavigate, useParams } from "react-router";
import {
  ArrowDownRight,
  ArrowLeft,
  ArrowUpRight,
  Loader2,
  Star,
} from "lucide-react";
import { toast } from "sonner";
import { formatPrice } from "../utils/currency";
import { isKycVerified } from "../utils/auth";
import { stockApi, watchlistApi, adminApi } from "../api";
import { Button } from "../components/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../components/ui/card";
import { StockChartPanel } from "../components/charts/StockChartPanel";
import type { StockOption } from "../types/ohlcv";

type StockEntity = {
  stock_id: string;
  symbol: string;
  name: string;
  price: number;
  open: number;
  high: number;
  low: number;
  previous_close: number;
  quantity: number;
  change: number;
  change_percent: number;
};

export default function StockDetails() {
  const { symbol } = useParams<{ symbol: string }>();
  const navigate = useNavigate();

  const [loading, setLoading] = useState(true);
  const [stock, setStock] = useState<StockEntity | null>(null);
  const [stockOptions, setStockOptions] = useState<StockOption[]>([]);
  const [marketOpen, setMarketOpen] = useState<boolean>(false);
  const marketOpenRef = useRef(false);

  const [inWatchlist, setInWatchlist] = useState(false);
  const [watchlistItemId, setWatchlistItemId] = useState<string | null>(null);
  const [watchlistLoading, setWatchlistLoading] = useState(false);

  const selectedSymbol = (symbol || stock?.symbol || "").toUpperCase();

  const loadStock = async () => {
    if (!symbol) return;
    try {
      const [searchRes, allStocksRes, watchRes] = await Promise.all([
        stockApi.search(symbol),
        stockApi.getAll(),
        watchlistApi.get().catch(() => ({ watchlist: [] })),
      ]);

      const allStocks: StockEntity[] = allStocksRes?.stocks || [];
      const target = (searchRes?.stocks || []).find(
        (s: StockEntity) => s.symbol.toUpperCase() === symbol.toUpperCase(),
      );

      if (!target) {
        setStock(null);
        return;
      }

      setStock(target);
      setStockOptions(
        allStocks.map((s) => ({
          stockId: s.stock_id,
          symbol: s.symbol,
          name: s.name,
        })),
      );

      const watchlist = watchRes?.watchlist || [];
      const existing = watchlist.find(
        (w: any) => w.stock_id === target.stock_id,
      );
      setInWatchlist(Boolean(existing));
      setWatchlistItemId(existing?.watchlist_id ?? null);
    } catch {
      toast.error("Failed to load stock details");
    }
  };

  useEffect(() => {
    if (!symbol) return;
    setLoading(true);
    loadStock().finally(() => setLoading(false));
  }, [symbol]);

  // Listen for market status and real-time price updates
  useEffect(() => {
    // Fetch initial market status
    const fetchMarketStatus = async () => {
      try {
        const status = await adminApi.getMarketStatus();
        if (status?.is_open !== undefined) {
          setMarketOpen(status.is_open);
          marketOpenRef.current = status.is_open;
          console.log(
            status.is_open
              ? "✅ Market OPEN on page load"
              : "🔒 Market CLOSED on page load"
          );
        }
      } catch (err) {
        console.log("Failed to fetch market status on load", err);
        // Default to true if fetch fails
        setMarketOpen(true);
        marketOpenRef.current = true;
      }
    };

    fetchMarketStatus();

    const connectMarketListener = () => {
      try {
        const protocol = window.location.protocol === "https:" ? "wss" : "ws";
        const backendUrl = import.meta.env.VITE_BACKEND_URL || "localhost:8080";
        const ws = new WebSocket(
          `${protocol}://${backendUrl}/ws/stocks`,
        );

        ws.onmessage = (event) => {
          try {
            const payload = JSON.parse(event.data) as {
              stocks?: any[];
              ticks?: any[];
              market_open?: boolean;
              type?: string;
            };

            // Handle market status updates
            if (payload.type === "market_status" && payload.market_open !== undefined) {
              setMarketOpen(payload.market_open as boolean);
              marketOpenRef.current = payload.market_open as boolean;  // Update ref immediately
              console.log(payload.market_open ? "🔓 Market opened" : "🔒 Market closed");
              return;
            }

            // Check if market is open using ref (synchronous check)
            if (!marketOpenRef.current) {
              console.debug("📊 Market closed, freezing prices");
              return;
            }

            // Update stock price in header if this is our symbol
            if (stock && (payload.stocks || payload.ticks)) {
              const stocksArray = payload.stocks || payload.ticks;
              const updatedStock = stocksArray?.find((s: any) =>
                (s.symbol || s.Symbol || "").toUpperCase() === stock.symbol.toUpperCase(),
              );

              if (updatedStock && typeof updatedStock.price === "number") {
                setStock((prev) => {
                  if (!prev) return prev;
                  const newPrice = updatedStock.price;
                  const change = newPrice - prev.previous_close;
                  const changePercent =
                    prev.previous_close !== 0
                      ? (change / prev.previous_close) * 100
                      : 0;

                  return {
                    ...prev,
                    price: newPrice,
                    change,
                    change_percent: changePercent,
                  };
                });
              }
            }
          } catch {
            // Ignore parse errors
          }
        };

        return () => {
          if (ws.readyState === WebSocket.OPEN) {
            ws.close();
          }
        };
      } catch {
        return () => {};
      }
    };

    const cleanup = connectMarketListener();
    return cleanup;
  }, [stock?.symbol, stock?.previous_close]);

  const marketCap = useMemo(() => {
    if (!stock) return 0;
    return stock.price * Math.max(stock.quantity || 0, 1);
  }, [stock]);

  const onSelectSymbol = (nextSymbol: string) => {
    navigate(`/stock/${nextSymbol}`);
  };

  const toggleWatchlist = async () => {
    if (!stock) return;
    setWatchlistLoading(true);
    try {
      if (inWatchlist && watchlistItemId) {
        await watchlistApi.remove(watchlistItemId);
        setInWatchlist(false);
        setWatchlistItemId(null);
        toast.success(`Removed ${stock.symbol} from watchlist`);
      } else {
        const res = await watchlistApi.add({ stock_id: stock.stock_id });
        setInWatchlist(true);
        if (res?.watchlist_id) setWatchlistItemId(res.watchlist_id);
        toast.success(`Added ${stock.symbol} to watchlist`);
      }
    } catch (err: any) {
      toast.error(err.message || "Failed to update watchlist");
    } finally {
      setWatchlistLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="flex h-full items-center justify-center">
        <Loader2 className="h-10 w-10 animate-spin text-primary" />
      </div>
    );
  }

  if (!stock) {
    return (
      <div className="flex h-full flex-col items-center justify-center space-y-4">
        <h2 className="text-2xl font-bold">Stock not found</h2>
        <Button onClick={() => navigate("/market")}>Back to Market</Button>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-7xl space-y-6">
      {!marketOpen && (
        <div className="bg-red-900/20 border border-red-500/50 rounded-lg p-4 text-center">
          <p className="text-red-400 font-semibold text-lg">🔒 Market Closed</p>
          <p className="text-red-300 text-sm mt-1">All prices are frozen. Live updates will resume when market opens.</p>
        </div>
      )}
      <div className="flex flex-col justify-between gap-4 md:flex-row md:items-start">
        <div>
          <Button
            variant="ghost"
            size="sm"
            className="-ml-2 mb-4 text-muted-foreground hover:bg-accent hover:text-foreground"
            onClick={() => navigate(-1)}
          >
            <ArrowLeft className="mr-2 h-4 w-4" />
            Back
          </Button>

          <div className="flex items-center gap-3">
            <h1 className="text-4xl font-bold">{stock.symbol}</h1>
            <Button
              variant="ghost"
              size="icon"
              onClick={toggleWatchlist}
              disabled={watchlistLoading}
              className={
                inWatchlist
                  ? "text-yellow-500 hover:bg-accent hover:text-yellow-400"
                  : "text-muted-foreground hover:bg-accent hover:text-foreground"
              }
            >
              {watchlistLoading ? (
                <Loader2 className="h-5 w-5 animate-spin" />
              ) : (
                <Star
                  className={`h-5 w-5 ${inWatchlist ? "fill-yellow-500" : ""}`}
                />
              )}
            </Button>
          </div>
          <p className="mt-1 text-muted-foreground">{stock.name}</p>
        </div>

        <div className="text-right">
          <p className="text-4xl font-bold">{formatPrice(stock.price)}</p>
          <div
            className={`mt-1 flex items-center justify-end gap-2 text-lg font-semibold ${
              stock.change >= 0 ? "text-green-500" : "text-red-500"
            }`}
          >
            {stock.change >= 0 ? (
              <ArrowUpRight className="h-5 w-5" />
            ) : (
              <ArrowDownRight className="h-5 w-5" />
            )}
            <span>
              {stock.change >= 0 ? "+" : ""}
              {stock.change.toFixed(2)} (
              {Math.abs(stock.change_percent).toFixed(2)}%)
            </span>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <div className="space-y-6 lg:col-span-2">
          <Card>
            <CardHeader>
              <CardTitle>Live OHLCV Charts</CardTitle>
            </CardHeader>
            <CardContent>
              <StockChartPanel
                options={
                  stockOptions.length > 0
                    ? stockOptions
                    : [
                        {
                          stockId: stock.stock_id,
                          symbol: stock.symbol,
                          name: stock.name,
                        },
                      ]
                }
                selectedSymbol={selectedSymbol}
                onSymbolChange={onSelectSymbol}
                basePrice={stock.price}
              />{" "}
            </CardContent>
          </Card>
        </div>

        <div className="space-y-6">
          <Card>
            <CardContent className="p-6">
              <div className="space-y-4">
                <Button
                  className="h-14 w-full bg-green-600 text-lg font-bold text-white shadow-lg hover:bg-green-700"
                  onClick={() => {
                    if (!isKycVerified()) {
                      toast.error("KYC verification required to trade");
                      navigate("/profile");
                      return;
                    }
                    navigate("/trade", {
                      state: { symbol: stock.symbol, type: "buy" },
                    });
                  }}
                >
                  Buy {stock.symbol}
                </Button>
                <Button
                  className="h-14 w-full bg-red-600 text-lg font-bold text-white shadow-lg hover:bg-red-700"
                  onClick={() => {
                    if (!isKycVerified()) {
                      toast.error("KYC verification required to trade");
                      navigate("/profile");
                      return;
                    }
                    navigate("/trade", {
                      state: { symbol: stock.symbol, type: "sell" },
                    });
                  }}
                >
                  Sell {stock.symbol}
                </Button>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Key Statistics</CardTitle>
            </CardHeader>
            <CardContent className="grid grid-cols-2 gap-4 text-sm">
              <div>
                <p className="text-muted-foreground">Open</p>
                <p className="font-semibold">{formatPrice(stock.open)}</p>
              </div>
              <div>
                <p className="text-muted-foreground">High</p>
                <p className="font-semibold">{formatPrice(stock.high)}</p>
              </div>
              <div>
                <p className="text-muted-foreground">Low</p>
                <p className="font-semibold">{formatPrice(stock.low)}</p>
              </div>
              <div>
                <p className="text-muted-foreground">Prev Close</p>
                <p className="font-semibold">
                  {formatPrice(stock.previous_close)}
                </p>
              </div>
              <div>
                <p className="text-muted-foreground">Volume</p>
                <p className="font-semibold">
                  {stock.quantity.toLocaleString()}
                </p>
              </div>
              <div>
                <p className="text-muted-foreground">Mkt Cap</p>
                <p className="font-semibold">
                  {formatPrice(marketCap / 1_000_000_000)}B
                </p>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
