import { useState, useEffect } from "react";
import { useParams, useNavigate } from "react-router";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../components/ui/card";
import { Button } from "../components/ui/button";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "../components/ui/tabs";
import {
  ArrowUpRight,
  ArrowDownRight,
  Star,
  TrendingUp,
  BarChart3,
  ArrowLeft,
  Loader2,
} from "lucide-react";
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from "recharts";
import { stockApi, watchlistApi } from "../api";
import { toast } from "sonner";

// Helper to generate a realistic looking curve since we don't have historical data yet
const generateMockChartData = (basePrice: number, period: string) => {
  const data = [];
  const points =
    period === "1D" ? 24 : period === "5D" ? 30 : period === "1M" ? 30 : 60;
  let currentPrice = basePrice * (0.95 + Math.random() * 0.05); // Start slightly off
  const volatility = period === "1D" ? 0.002 : period === "5D" ? 0.005 : 0.015;

  for (let i = 0; i < points; i++) {
    const change = currentPrice * volatility * (Math.random() - 0.45);
    currentPrice += change;

    let timeLabel = `Point ${i}`;
    if (period === "1D") {
      timeLabel = `${9 + Math.floor(i / 4)}:${(i % 4) * 15 || "00"}`;
    } else {
      timeLabel = `Day ${i + 1}`;
    }

    data.push({
      time: timeLabel,
      price: parseFloat(currentPrice.toFixed(2)),
    });
  }
  // Ensure the last point matches the real current price exactly
  data.push({ time: "Now", price: basePrice });
  return data;
};

export default function StockDetails() {
  const { symbol } = useParams<{ symbol: string }>();
  const navigate = useNavigate();
  const [stock, setStock] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [chartPeriod, setChartPeriod] = useState<
    "1D" | "5D" | "1M" | "6M" | "1Y"
  >("1D");
  const [chartData, setChartData] = useState<any[]>([]);

  const [inWatchlist, setInWatchlist] = useState(false);
  const [watchlistItemId, setWatchlistItemId] = useState<string | null>(null);
  const [watchlistLoading, setWatchlistLoading] = useState(false);

  useEffect(() => {
    const fetchStockDetails = async () => {
      if (!symbol) return;
      try {
        setLoading(true);
        // 1. Fetch stock details
        const res = await stockApi.search(symbol);
        if (res && res.stocks && res.stocks.length > 0) {
          const match = res.stocks.find(
            (s: any) => s.symbol.toUpperCase() === symbol.toUpperCase(),
          );

          if (match) {
            // Enrich basic stock data with pseudo-data for display purposes
            // (Since the current backend schema only has price/quantity)
            const pseudoChange = ((match.symbol.length * 7) % 10) - 5;
            const enrichedStock = {
              ...match,
              change: pseudoChange || 1.25,
              changePercent: (pseudoChange / match.price) * 100 || 0.8,
              open: match.price - pseudoChange,
              high: match.price * 1.02,
              low: match.price * 0.98,
              volume: match.quantity || 1500000,
              marketCap: match.price * (match.quantity || 10000000),
              previousClose: match.price - pseudoChange * 1.5,
              weekHigh52: match.price * 1.4,
              weekLow52: match.price * 0.7,
            };

            setStock(enrichedStock);
            setChartData(
              generateMockChartData(enrichedStock.price, chartPeriod),
            );

            // 2. Check if in watchlist
            const wlRes = await watchlistApi
              .get()
              .catch(() => ({ watchlist: [] }));
            const wl = wlRes?.watchlist || [];
            const existingWlItem = wl.find(
              (w: any) => w.stock_id === enrichedStock.stock_id,
            );
            if (existingWlItem) {
              setInWatchlist(true);
              setWatchlistItemId(existingWlItem.watchlist_id);
            }
          }
        }
      } catch (e) {
        toast.error("Failed to load stock details");
      } finally {
        setLoading(false);
      }
    };

    fetchStockDetails();
  }, [symbol]);

  useEffect(() => {
    if (stock) {
      setChartData(generateMockChartData(stock.price, chartPeriod));
    }
  }, [chartPeriod, stock]);

  const toggleWatchlist = async () => {
    if (!stock) return;
    try {
      setWatchlistLoading(true);
      if (inWatchlist && watchlistItemId) {
        await watchlistApi.remove(watchlistItemId);
        setInWatchlist(false);
        setWatchlistItemId(null);
        toast.success(`Removed ${stock.symbol} from watchlist`);
      } else {
        const res = await watchlistApi.add({ stock_id: stock.stock_id });
        setInWatchlist(true);
        // Assuming the backend returns the created object or id, if not we'll just optimistically set true
        if (res && res.watchlist_id) {
          setWatchlistItemId(res.watchlist_id);
        }
        toast.success(`Added ${stock.symbol} to watchlist`);
      }
    } catch (e: any) {
      toast.error("Failed to update watchlist");
      // Fallback reload just in case
    } finally {
      setWatchlistLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="flex justify-center items-center h-full text-white">
        <Loader2 className="w-10 h-10 animate-spin text-blue-500" />
      </div>
    );
  }

  if (!stock) {
    return (
      <div className="flex flex-col items-center justify-center h-full space-y-4">
        <h2 className="text-2xl font-bold text-white">Stock not found</h2>
        <p className="text-gray-300">We couldn't find data for {symbol}</p>
        <Button onClick={() => navigate("/market")}>Back to Market</Button>
      </div>
    );
  }

  return (
    <div className="max-w-6xl mx-auto space-y-6 text-white">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-start justify-between gap-4">
        <div>
          <Button
            variant="ghost"
            size="sm"
            className="mb-4 text-gray-400 hover:text-white hover:bg-gray-800 -ml-2"
            onClick={() => navigate(-1)}
          >
            <ArrowLeft className="w-4 h-4 mr-2" />
            Back
          </Button>
          <div className="flex items-center gap-3">
            <h1 className="text-4xl font-bold text-white">{stock.symbol}</h1>
            <Button
              variant="ghost"
              size="icon"
              className={
                inWatchlist
                  ? "text-yellow-500 hover:text-yellow-600 hover:bg-gray-800"
                  : "text-gray-400 hover:text-white hover:bg-gray-800"
              }
              onClick={toggleWatchlist}
              disabled={watchlistLoading}
            >
              {watchlistLoading ? (
                <Loader2 className="w-5 h-5 animate-spin" />
              ) : (
                <Star
                  className={`w-5 h-5 ${inWatchlist ? "fill-yellow-500" : ""}`}
                />
              )}
            </Button>
          </div>
          <p className="text-gray-300 mt-1">{stock.name}</p>
        </div>

        <div className="text-right">
          <p className="text-4xl font-bold text-white">
            ${stock.price.toFixed(2)}
          </p>
          <div
            className={`flex items-center justify-end gap-2 mt-1 text-lg font-semibold ${
              stock.change >= 0 ? "text-green-500" : "text-red-500"
            }`}
          >
            {stock.change >= 0 ? (
              <ArrowUpRight className="w-5 h-5" />
            ) : (
              <ArrowDownRight className="w-5 h-5" />
            )}
            <span>
              {stock.change >= 0 ? "+" : ""}
              {stock.change.toFixed(2)} (
              {Math.abs(stock.changePercent).toFixed(2)}%)
            </span>
          </div>
        </div>
      </div>

      {/* Main Content */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="lg:col-span-2 space-y-6">
          {/* Chart Card */}
          <Card className="text-white bg-card">
            <CardHeader className="flex flex-row items-center justify-between pb-2 border-b border-gray-800">
              <CardTitle className="text-white">Price History</CardTitle>
              <Tabs
                value={chartPeriod}
                onValueChange={(v) => setChartPeriod(v as any)}
              >
                <TabsList className="bg-gray-900 border border-gray-800">
                  <TabsTrigger
                    value="1D"
                    className="data-[state=active]:bg-gray-800 data-[state=active]:text-white"
                  >
                    1D
                  </TabsTrigger>
                  <TabsTrigger
                    value="5D"
                    className="data-[state=active]:bg-gray-800 data-[state=active]:text-white"
                  >
                    5D
                  </TabsTrigger>
                  <TabsTrigger
                    value="1M"
                    className="data-[state=active]:bg-gray-800 data-[state=active]:text-white"
                  >
                    1M
                  </TabsTrigger>
                  <TabsTrigger
                    value="6M"
                    className="data-[state=active]:bg-gray-800 data-[state=active]:text-white"
                  >
                    6M
                  </TabsTrigger>
                  <TabsTrigger
                    value="1Y"
                    className="data-[state=active]:bg-gray-800 data-[state=active]:text-white"
                  >
                    1Y
                  </TabsTrigger>
                </TabsList>
              </Tabs>
            </CardHeader>
            <CardContent className="pt-6">
              <div className="h-[400px] w-full">
                <ResponsiveContainer width="100%" height="100%">
                  <AreaChart data={chartData}>
                    <defs>
                      <linearGradient
                        id="colorPrice"
                        x1="0"
                        y1="0"
                        x2="0"
                        y2="1"
                      >
                        <stop
                          offset="5%"
                          stopColor={stock.change >= 0 ? "#10b981" : "#ef4444"}
                          stopOpacity={0.3}
                        />
                        <stop
                          offset="95%"
                          stopColor={stock.change >= 0 ? "#10b981" : "#ef4444"}
                          stopOpacity={0}
                        />
                      </linearGradient>
                    </defs>
                    <CartesianGrid
                      strokeDasharray="3 3"
                      stroke="#374151"
                      vertical={false}
                    />
                    <XAxis
                      dataKey="time"
                      stroke="#9ca3af"
                      tick={{ fill: "#9ca3af" }}
                      tickMargin={10}
                      minTickGap={30}
                    />
                    <YAxis
                      domain={["auto", "auto"]}
                      stroke="#9ca3af"
                      tick={{ fill: "#9ca3af" }}
                      tickFormatter={(value) => `$${value}`}
                      width={60}
                    />
                    <Tooltip
                      cursor={{
                        stroke: "#4b5563",
                        strokeWidth: 1,
                        strokeDasharray: "3 3",
                        fill: "transparent",
                      }}
                      itemStyle={{ color: "white" }}
                      labelStyle={{ color: "gray", marginBottom: "4px" }}
                      contentStyle={{
                        backgroundColor: "#1f2937",
                        border: "1px solid #4b5563",
                        color: "white",
                        borderRadius: "8px",
                        boxShadow: "0 4px 6px -1px rgba(0, 0, 0, 0.5)",
                      }}
                      formatter={(value: number) => [
                        `$${value.toFixed(2)}`,
                        "Price",
                      ]}
                    />
                    <Area
                      type="monotone"
                      dataKey="price"
                      stroke={stock.change >= 0 ? "#10b981" : "#ef4444"}
                      strokeWidth={2}
                      fillOpacity={1}
                      fill="url(#colorPrice)"
                      isAnimationActive={false}
                    />
                  </AreaChart>
                </ResponsiveContainer>
              </div>
            </CardContent>
          </Card>

          {/* Key Statistics */}
          <Card className="text-white bg-card">
            <CardHeader>
              <CardTitle className="text-white flex items-center">
                <BarChart3 className="w-5 h-5 mr-2" />
                Key Statistics
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-2 md:grid-cols-3 gap-6">
                <div className="flex flex-col border-b border-gray-800 pb-3">
                  <span className="text-gray-400 text-sm mb-1">Open</span>
                  <span className="text-lg font-semibold text-white">
                    ${stock.open.toFixed(2)}
                  </span>
                </div>
                <div className="flex flex-col border-b border-gray-800 pb-3">
                  <span className="text-gray-400 text-sm mb-1">High</span>
                  <span className="text-lg font-semibold text-white">
                    ${stock.high.toFixed(2)}
                  </span>
                </div>
                <div className="flex flex-col border-b border-gray-800 pb-3">
                  <span className="text-gray-400 text-sm mb-1">Low</span>
                  <span className="text-lg font-semibold text-white">
                    ${stock.low.toFixed(2)}
                  </span>
                </div>
                <div className="flex flex-col">
                  <span className="text-gray-400 text-sm mb-1">Volume</span>
                  <span className="text-lg font-semibold text-white">
                    {(stock.volume / 1000000).toFixed(2)}M
                  </span>
                </div>
                <div className="flex flex-col">
                  <span className="text-gray-400 text-sm mb-1">Mkt Cap</span>
                  <span className="text-lg font-semibold text-white">
                    ${(stock.marketCap / 1000000000).toFixed(2)}B
                  </span>
                </div>
                <div className="flex flex-col">
                  <span className="text-gray-400 text-sm mb-1">Prev Close</span>
                  <span className="text-lg font-semibold text-white">
                    ${stock.previousClose.toFixed(2)}
                  </span>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>

        <div className="space-y-6">
          {/* Action Card */}
          <Card className="text-white bg-card">
            <CardContent className="p-6">
              <div className="space-y-4">
                <Button
                  className="w-full h-14 text-lg font-bold bg-green-600 hover:bg-green-700 text-white shadow-lg"
                  onClick={() =>
                    navigate("/trade", {
                      state: { symbol: stock.symbol, type: "buy" },
                    })
                  }
                >
                  Buy {stock.symbol}
                </Button>
                <Button
                  className="w-full h-14 text-lg font-bold bg-red-600 hover:bg-red-700 text-white shadow-lg"
                  onClick={() =>
                    navigate("/trade", {
                      state: { symbol: stock.symbol, type: "sell" },
                    })
                  }
                >
                  Sell {stock.symbol}
                </Button>
              </div>

              <div className="mt-6 pt-6 border-t border-gray-800 text-sm text-gray-400 text-center">
                Trading hours: 9:30 AM - 4:00 PM EST
              </div>
            </CardContent>
          </Card>

          {/* Order Book (Mocked Visual) */}
          <Card className="text-white bg-card">
            <CardHeader>
              <CardTitle className="text-white text-lg">Order Book</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <div className="text-xs text-gray-400 grid grid-cols-2 gap-2 pb-2 border-b border-gray-800">
                    <span>Bid</span>
                    <span className="text-right">Qty</span>
                  </div>
                  <div className="mt-2 space-y-1">
                    {[1, 2, 3, 4, 5].map((i) => (
                      <div
                        key={`bid-${i}`}
                        className="grid grid-cols-2 gap-2 text-sm"
                      >
                        <span className="text-green-500 font-medium">
                          ${(stock.price - i * 0.05).toFixed(2)}
                        </span>
                        <span className="text-right text-gray-300">
                          {Math.floor(Math.random() * 500) + 10}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
                <div>
                  <div className="text-xs text-gray-400 grid grid-cols-2 gap-2 pb-2 border-b border-gray-800">
                    <span>Ask</span>
                    <span className="text-right">Qty</span>
                  </div>
                  <div className="mt-2 space-y-1">
                    {[1, 2, 3, 4, 5].map((i) => (
                      <div
                        key={`ask-${i}`}
                        className="grid grid-cols-2 gap-2 text-sm"
                      >
                        <span className="text-red-500 font-medium">
                          ${(stock.price + i * 0.05).toFixed(2)}
                        </span>
                        <span className="text-right text-gray-300">
                          {Math.floor(Math.random() * 500) + 10}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
