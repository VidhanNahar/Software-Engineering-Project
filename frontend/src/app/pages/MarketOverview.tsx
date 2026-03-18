import { useState, useEffect } from "react";
import { useNavigate } from "react-router";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../components/ui/card";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "../components/ui/tabs";
import { ArrowUpRight, ArrowDownRight } from "lucide-react";
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Cell,
} from "recharts";
import { stockApi } from "../api";

export default function MarketOverview() {
  const navigate = useNavigate();
  const [activeTab, setActiveTab] = useState("overview");
  const [allStocks, setAllStocks] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchStocks = async () => {
      try {
        const response = await stockApi.getAll();
        if (response && response.stocks) {
          // Simulate change, changePercent and volume since backend currently just returns stock basics
          const enrichedStocks = response.stocks.map((s: any) => {
            // Pseudo-random generation based on stock properties to stay consistent during renders
            const pseudoRand = ((s.symbol.length * 13) % 10) - 5;
            const change = pseudoRand || 1.5;
            const changePercent = (change / s.price) * 100;
            return {
              ...s,
              change,
              changePercent,
              volume: s.quantity || Math.floor(Math.random() * 1000000),
            };
          });
          setAllStocks(enrichedStocks);
        }
      } catch (err) {
        console.error("Failed to fetch stocks", err);
      } finally {
        setLoading(false);
      }
    };
    fetchStocks();
  }, []);

  const gainers = [...allStocks]
    .sort((a, b) => b.changePercent - a.changePercent)
    .slice(0, 5);
  const losers = [...allStocks]
    .sort((a, b) => a.changePercent - b.changePercent)
    .slice(0, 5);
  const mostActive = [...allStocks]
    .sort((a, b) => b.volume - a.volume)
    .slice(0, 5);

  const sectorPerformance = [
    { sector: "Technology", change: 1.25, value: 156.8 },
    { sector: "Healthcare", change: 0.85, value: 142.3 },
    { sector: "Financial", change: -0.35, value: 128.5 },
    { sector: "Energy", change: 1.95, value: 118.2 },
    { sector: "Consumer", change: 0.45, value: 134.7 },
    { sector: "Industrial", change: -0.65, value: 121.4 },
  ];

  const marketIndices = [
    { name: "S&P 500", value: 5234.18, change: 45.67, percent: 0.87 },
    { name: "Dow Jones", value: 38905.66, change: 476.84, percent: 1.24 },
    { name: "NASDAQ", value: 16248.52, change: -12.34, percent: -0.08 },
    { name: "Russell 2000", value: 2065.88, change: 15.22, percent: 0.74 },
  ];

  const volumeData = [
    { time: "09:30", volume: 1200 },
    { time: "10:30", volume: 1800 },
    { time: "11:30", volume: 1500 },
    { time: "12:30", volume: 900 },
    { time: "13:30", volume: 1100 },
    { time: "14:30", volume: 1600 },
    { time: "15:30", volume: 2400 },
  ];

  if (loading) {
    return <div className="p-6 text-white">Loading market data...</div>;
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold text-white">Market Overview</h1>
        <p className="text-gray-300 mt-1">
          Real-time market data and analytics
        </p>
      </div>

      {/* Market Indices */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {marketIndices.map((index) => (
          <Card key={index.name}>
            <CardContent className="p-6">
              <p className="text-sm text-gray-300">{index.name}</p>
              <p className="text-2xl font-bold text-white mt-1">
                {index.value.toLocaleString("en-US", {
                  minimumFractionDigits: 2,
                })}
              </p>
              <div
                className={`flex items-center gap-1 mt-1 ${
                  index.change >= 0 ? "text-green-600" : "text-red-600"
                }`}
              >
                {index.change >= 0 ? (
                  <ArrowUpRight className="w-4 h-4" />
                ) : (
                  <ArrowDownRight className="w-4 h-4" />
                )}
                <span className="text-sm font-semibold">
                  {index.change >= 0 ? "+" : ""}
                  {index.change.toFixed(2)} ({index.percent >= 0 ? "+" : ""}
                  {index.percent.toFixed(2)}%)
                </span>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Main Charts */}
        <div className="lg:col-span-2 space-y-6">
          <Card className="text-white">
            <CardHeader>
              <CardTitle className="text-white">Sector Performance</CardTitle>
            </CardHeader>
            <CardContent>
              <ResponsiveContainer width="100%" height={300}>
                <BarChart data={sectorPerformance}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                  <XAxis dataKey="sector" stroke="#6b7280" />
                  <YAxis stroke="#6b7280" />
                  <Tooltip
                    cursor={{ fill: "transparent" }}
                    itemStyle={{ color: "white" }}
                    labelStyle={{ color: "white" }}
                    contentStyle={{
                      backgroundColor: "#1f2937",
                      border: "1px solid white",
                      color: "white",
                      borderRadius: "8px",
                    }}
                  />
                  <Bar
                    dataKey="change"
                    radius={[8, 8, 0, 0]}
                    activeBar={{ stroke: "white", strokeWidth: 2 }}
                  >
                    {sectorPerformance.map((entry) => (
                      <Cell
                        key={`cell-${entry.sector}`}
                        fill={entry.change >= 0 ? "#10b981" : "#ef4444"}
                      />
                    ))}
                  </Bar>
                </BarChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>

          {/* Market Activity */}
          <Card className="text-white">
            <CardHeader>
              <CardTitle className="text-white">Market Volume</CardTitle>
            </CardHeader>
            <CardContent>
              <ResponsiveContainer width="100%" height={250}>
                <BarChart data={volumeData}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                  <XAxis dataKey="time" stroke="#6b7280" />
                  <YAxis stroke="#6b7280" />
                  <Tooltip
                    cursor={{ fill: "transparent" }}
                    itemStyle={{ color: "white" }}
                    labelStyle={{ color: "white" }}
                    contentStyle={{
                      backgroundColor: "#1f2937",
                      border: "1px solid white",
                      color: "white",
                      borderRadius: "8px",
                    }}
                  />
                  <Bar
                    dataKey="volume"
                    fill="#3b82f6"
                    radius={[8, 8, 0, 0]}
                    activeBar={{ stroke: "white", strokeWidth: 2 }}
                  />
                </BarChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>

          <Card className="text-white">
            <CardHeader>
              <CardTitle className="text-white">Market Breadth</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm text-gray-300">Advancing (45%)</span>
                <span className="text-sm text-gray-300">Declining (55%)</span>
              </div>
              <div className="w-full h-4 flex rounded-full overflow-hidden">
                <div className="bg-green-500 h-full" style={{ width: "45%" }} />
                <div className="bg-red-500 h-full" style={{ width: "55%" }} />
              </div>
              <div className="flex justify-between mt-4">
                <div className="text-center">
                  <p className="text-2xl font-bold text-green-500">3,245</p>
                  <p className="text-sm text-gray-300">Issues Advancing</p>
                </div>
                <div className="text-center">
                  <p className="text-2xl font-bold text-red-500">3,892</p>
                  <p className="text-sm text-gray-300">Issues Declining</p>
                </div>
                <div className="text-center">
                  <p className="text-2xl font-bold text-gray-500">214</p>
                  <p className="text-sm text-gray-300">Unchanged</p>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Top Lists */}
        <div className="space-y-6">
          <Tabs
            value={activeTab}
            onValueChange={setActiveTab}
            className="w-full"
          >
            <TabsList className="grid w-full grid-cols-3">
              <TabsTrigger value="overview">Gainers</TabsTrigger>
              <TabsTrigger value="details">Losers</TabsTrigger>
              <TabsTrigger value="financials">Active</TabsTrigger>
            </TabsList>

            <TabsContent value="overview">
              <Card className="text-white">
                <CardContent className="pt-6">
                  <div className="space-y-4">
                    {gainers.map((stock) => (
                      <div
                        key={stock.symbol}
                        className="flex items-center justify-between p-4 border border-transparent hover:border-white hover:bg-transparent rounded-lg cursor-pointer transition-colors"
                        onClick={() => navigate(`/stock/${stock.symbol}`)}
                      >
                        <div>
                          <p className="font-semibold text-white">
                            {stock.symbol}
                          </p>
                          <p className="text-sm text-gray-300">{stock.name}</p>
                        </div>
                        <div className="text-right">
                          <p className="font-semibold text-white">
                            ${stock.price.toFixed(2)}
                          </p>
                          <p className="text-sm text-green-500 flex items-center justify-end gap-1">
                            <ArrowUpRight className="w-4 h-4" />
                            {stock.changePercent.toFixed(2)}%
                          </p>
                        </div>
                      </div>
                    ))}
                  </div>
                </CardContent>
              </Card>
            </TabsContent>

            <TabsContent value="details">
              <Card className="text-white">
                <CardContent className="pt-6">
                  <div className="space-y-4">
                    {losers.map((stock) => (
                      <div
                        key={stock.symbol}
                        className="flex items-center justify-between p-4 border border-transparent hover:border-white hover:bg-transparent rounded-lg cursor-pointer transition-colors"
                        onClick={() => navigate(`/stock/${stock.symbol}`)}
                      >
                        <div>
                          <p className="font-semibold text-white">
                            {stock.symbol}
                          </p>
                          <p className="text-sm text-gray-300">{stock.name}</p>
                        </div>
                        <div className="text-right">
                          <p className="font-semibold text-white">
                            ${stock.price.toFixed(2)}
                          </p>
                          <p className="text-sm text-red-500 flex items-center justify-end gap-1">
                            <ArrowDownRight className="w-4 h-4" />
                            {Math.abs(stock.changePercent).toFixed(2)}%
                          </p>
                        </div>
                      </div>
                    ))}
                  </div>
                </CardContent>
              </Card>
            </TabsContent>

            <TabsContent value="financials">
              <Card className="text-white">
                <CardContent className="pt-6">
                  <div className="space-y-4">
                    {mostActive.map((stock) => (
                      <div
                        key={stock.symbol}
                        className="flex items-center justify-between p-4 border border-transparent hover:border-white hover:bg-transparent rounded-lg cursor-pointer transition-colors"
                        onClick={() => navigate(`/stock/${stock.symbol}`)}
                      >
                        <div>
                          <p className="font-semibold text-white">
                            {stock.symbol}
                          </p>
                          <p className="text-sm text-gray-300">{stock.name}</p>
                        </div>
                        <div className="text-right">
                          <p className="font-semibold text-white">
                            ${stock.price.toFixed(2)}
                          </p>
                          <p className="text-sm text-gray-300">
                            Volume: {stock.volume.toLocaleString()}
                          </p>
                        </div>
                      </div>
                    ))}
                  </div>
                </CardContent>
              </Card>
            </TabsContent>
          </Tabs>
        </div>
      </div>
    </div>
  );
}
