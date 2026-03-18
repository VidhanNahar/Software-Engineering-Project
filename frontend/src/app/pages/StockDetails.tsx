import { useState, useEffect } from "react";
import { useParams, useNavigate } from "react-router";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../components/ui/tabs";
import {
  ArrowUpRight,
  ArrowDownRight,
  Star,
  TrendingUp,
  BarChart3,
  ArrowLeft,
} from "lucide-react";
import {
  LineChart,
  Line,
  AreaChart,
  Area,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from "recharts";
import { mockStocks, generateChartData, generateIntradayData, simulatePriceUpdate } from "../utils/mockData";
import { Stock } from "../types/stock";
import { toast } from "sonner";

export default function StockDetails() {
  const { symbol } = useParams<{ symbol: string }>();
  const navigate = useNavigate();
  const [stock, setStock] = useState<Stock | null>(null);
  const [chartPeriod, setChartPeriod] = useState<"1D" | "5D" | "1M" | "6M" | "1Y">("1D");
  const [chartData, setChartData] = useState<any[]>([]);

  useEffect(() => {
    if (symbol && mockStocks[symbol]) {
      setStock(mockStocks[symbol]);
      updateChartData(symbol, chartPeriod);
    }
  }, [symbol, chartPeriod]);

  useEffect(() => {
    if (!stock) return;

    const interval = setInterval(() => {
      setStock((prev) => {
        if (!prev) return prev;
        const newPrice = simulatePriceUpdate(prev.price);
        const change = newPrice - prev.previousClose;
        const changePercent = (change / prev.previousClose) * 100;
        return {
          ...prev,
          price: newPrice,
          change: parseFloat(change.toFixed(2)),
          changePercent: parseFloat(changePercent.toFixed(2)),
        };
      });
    }, 2000);

    return () => clearInterval(interval);
  }, [stock]);

  const updateChartData = (sym: string, period: string) => {
    const basePrice = mockStocks[sym].price;
    if (period === "1D") {
      setChartData(generateIntradayData(basePrice));
    } else {
      const days = period === "5D" ? 5 : period === "1M" ? 30 : period === "6M" ? 180 : 365;
      setChartData(generateChartData(basePrice, days));
    }
  };

  if (!stock || !symbol) {
    return (
      <div className="flex items-center justify-center h-96">
        <p className="text-gray-600">Stock not found</p>
      </div>
    );
  }

  const orderBook = {
    bids: [
      { price: stock.price - 0.05, quantity: 1500, total: (stock.price - 0.05) * 1500 },
      { price: stock.price - 0.10, quantity: 2300, total: (stock.price - 0.10) * 2300 },
      { price: stock.price - 0.15, quantity: 1800, total: (stock.price - 0.15) * 1800 },
      { price: stock.price - 0.20, quantity: 3200, total: (stock.price - 0.20) * 3200 },
      { price: stock.price - 0.25, quantity: 2100, total: (stock.price - 0.25) * 2100 },
    ],
    asks: [
      { price: stock.price + 0.05, quantity: 1200, total: (stock.price + 0.05) * 1200 },
      { price: stock.price + 0.10, quantity: 1900, total: (stock.price + 0.10) * 1900 },
      { price: stock.price + 0.15, quantity: 2400, total: (stock.price + 0.15) * 2400 },
      { price: stock.price + 0.20, quantity: 1600, total: (stock.price + 0.20) * 1600 },
      { price: stock.price + 0.25, quantity: 2800, total: (stock.price + 0.25) * 2800 },
    ],
  };

  return (
    <div className="space-y-6">
      {/* Back Button */}
      <Button variant="ghost" onClick={() => navigate(-1)}>
        <ArrowLeft className="w-4 h-4 mr-2" />
        Back
      </Button>

      {/* Stock Header */}
      <div className="flex items-start justify-between">
        <div>
          <div className="flex items-center gap-3">
            <h1 className="text-4xl font-bold text-gray-900">{stock.symbol}</h1>
            <Button variant="ghost" size="icon">
              <Star className="w-5 h-5 text-yellow-500 fill-yellow-500" />
            </Button>
          </div>
          <p className="text-gray-600 mt-1">{stock.name}</p>
        </div>
        <div className="text-right">
          <p className="text-4xl font-bold text-gray-900">${stock.price.toFixed(2)}</p>
          <div
            className={`flex items-center gap-2 justify-end mt-1 ${
              stock.change >= 0 ? "text-green-600" : "text-red-600"
            }`}
          >
            {stock.change >= 0 ? (
              <ArrowUpRight className="w-5 h-5" />
            ) : (
              <ArrowDownRight className="w-5 h-5" />
            )}
            <span className="text-xl font-semibold">
              {stock.change >= 0 ? "+" : ""}${stock.change.toFixed(2)} (
              {stock.changePercent >= 0 ? "+" : ""}
              {stock.changePercent.toFixed(2)}%)
            </span>
          </div>
        </div>
      </div>

      {/* Stock Stats */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <Card>
          <CardContent className="p-4">
            <p className="text-sm text-gray-600">Open</p>
            <p className="text-xl font-semibold text-gray-900 mt-1">
              ${stock.open.toFixed(2)}
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <p className="text-sm text-gray-600">High</p>
            <p className="text-xl font-semibold text-gray-900 mt-1">
              ${stock.high.toFixed(2)}
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <p className="text-sm text-gray-600">Low</p>
            <p className="text-xl font-semibold text-gray-900 mt-1">
              ${stock.low.toFixed(2)}
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <p className="text-sm text-gray-600">Volume</p>
            <p className="text-xl font-semibold text-gray-900 mt-1">
              {(stock.volume / 1000000).toFixed(2)}M
            </p>
          </CardContent>
        </Card>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Chart */}
        <div className="lg:col-span-2">
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle>Price Chart</CardTitle>
                <div className="flex gap-2">
                  {(["1D", "5D", "1M", "6M", "1Y"] as const).map((period) => (
                    <Button
                      key={period}
                      variant={chartPeriod === period ? "default" : "outline"}
                      size="sm"
                      onClick={() => setChartPeriod(period)}
                    >
                      {period}
                    </Button>
                  ))}
                </div>
              </div>
            </CardHeader>
            <CardContent>
              <ResponsiveContainer width="100%" height={400}>
                <AreaChart data={chartData}>
                  <defs>
                    <linearGradient id="colorPrice" x1="0" y1="0" x2="0" y2="1">
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
                  <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                  <XAxis dataKey="time" stroke="#6b7280" />
                  <YAxis domain={["auto", "auto"]} stroke="#6b7280" />
                  <Tooltip
                    contentStyle={{
                      backgroundColor: "#fff",
                      border: "1px solid #e5e7eb",
                      borderRadius: "8px",
                    }}
                  />
                  <Area
                    type="monotone"
                    dataKey="price"
                    stroke={stock.change >= 0 ? "#10b981" : "#ef4444"}
                    strokeWidth={2}
                    fill="url(#colorPrice)"
                  />
                </AreaChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>

          {/* Trading Actions */}
          <Card className="mt-6">
            <CardHeader>
              <CardTitle>Trade {stock.symbol}</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-2 gap-4">
                <Button
                  className="h-16 bg-green-600 hover:bg-green-700"
                  onClick={() => {
                    navigate("/trade", { state: { symbol: stock.symbol, type: "buy" } });
                  }}
                >
                  <TrendingUp className="w-5 h-5 mr-2" />
                  Buy {stock.symbol}
                </Button>
                <Button
                  variant="destructive"
                  className="h-16"
                  onClick={() => {
                    navigate("/trade", { state: { symbol: stock.symbol, type: "sell" } });
                  }}
                >
                  <BarChart3 className="w-5 h-5 mr-2" />
                  Sell {stock.symbol}
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Order Book & Stats */}
        <div className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Market Stats</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex justify-between">
                <span className="text-gray-600">Market Cap</span>
                <span className="font-semibold">
                  ${(stock.marketCap / 1000000000000).toFixed(2)}T
                </span>
              </div>
              <div className="flex justify-between">
                <span className="text-gray-600">Prev Close</span>
                <span className="font-semibold">${stock.previousClose.toFixed(2)}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-gray-600">Volume</span>
                <span className="font-semibold">
                  {(stock.volume / 1000000).toFixed(2)}M
                </span>
              </div>
              <div className="flex justify-between">
                <span className="text-gray-600">52W High</span>
                <span className="font-semibold">
                  ${(stock.price * 1.15).toFixed(2)}
                </span>
              </div>
              <div className="flex justify-between">
                <span className="text-gray-600">52W Low</span>
                <span className="font-semibold">
                  ${(stock.price * 0.75).toFixed(2)}
                </span>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Order Book</CardTitle>
            </CardHeader>
            <CardContent>
              <Tabs defaultValue="bids">
                <TabsList className="grid w-full grid-cols-2">
                  <TabsTrigger value="bids">Bids</TabsTrigger>
                  <TabsTrigger value="asks">Asks</TabsTrigger>
                </TabsList>
                <TabsContent value="bids" className="space-y-2 mt-4">
                  <div className="text-xs text-gray-600 grid grid-cols-3 gap-2 pb-2 border-b">
                    <span>Price</span>
                    <span className="text-right">Quantity</span>
                    <span className="text-right">Total</span>
                  </div>
                  {orderBook.bids.map((bid, idx) => (
                    <div key={idx} className="grid grid-cols-3 gap-2 text-sm">
                      <span className="text-green-600 font-medium">
                        ${bid.price.toFixed(2)}
                      </span>
                      <span className="text-right text-gray-700">{bid.quantity}</span>
                      <span className="text-right text-gray-700">
                        ${bid.total.toFixed(0)}
                      </span>
                    </div>
                  ))}
                </TabsContent>
                <TabsContent value="asks" className="space-y-2 mt-4">
                  <div className="text-xs text-gray-600 grid grid-cols-3 gap-2 pb-2 border-b">
                    <span>Price</span>
                    <span className="text-right">Quantity</span>
                    <span className="text-right">Total</span>
                  </div>
                  {orderBook.asks.map((ask, idx) => (
                    <div key={idx} className="grid grid-cols-3 gap-2 text-sm">
                      <span className="text-red-600 font-medium">
                        ${ask.price.toFixed(2)}
                      </span>
                      <span className="text-right text-gray-700">{ask.quantity}</span>
                      <span className="text-right text-gray-700">
                        ${ask.total.toFixed(0)}
                      </span>
                    </div>
                  ))}
                </TabsContent>
              </Tabs>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
