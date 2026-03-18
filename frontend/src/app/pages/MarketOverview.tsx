import { useState } from "react";
import { useNavigate } from "react-router";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../components/ui/tabs";
import { ArrowUpRight, ArrowDownRight, TrendingUp, Activity } from "lucide-react";
import {
  BarChart,
  Bar,
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Cell,
} from "recharts";
import { mockStocks } from "../utils/mockData";

export default function MarketOverview() {
  const navigate = useNavigate();
  const [activeTab, setActiveTab] = useState("overview");

  const allStocks = Object.values(mockStocks);
  const gainers = [...allStocks].sort((a, b) => b.changePercent - a.changePercent).slice(0, 5);
  const losers = [...allStocks].sort((a, b) => a.changePercent - b.changePercent).slice(0, 5);
  const mostActive = [...allStocks].sort((a, b) => b.volume - a.volume).slice(0, 5);

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
    { name: "NASDAQ", value: 16315.70, change: -37.56, percent: -0.23 },
    { name: "Russell 2000", value: 2089.32, change: 12.45, percent: 0.60 },
  ];

  const volumeData = [
    { time: "9:30", volume: 45 },
    { time: "10:00", volume: 78 },
    { time: "11:00", volume: 52 },
    { time: "12:00", volume: 34 },
    { time: "1:00", volume: 41 },
    { time: "2:00", volume: 89 },
    { time: "3:00", volume: 125 },
    { time: "4:00", volume: 156 },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold text-white">Market Overview</h1>
        <p className="text-gray-300 mt-1">Real-time market data and analytics</p>
      </div>

      {/* Market Indices */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {marketIndices.map((index) => (
          <Card key={index.name}>
            <CardContent className="p-6">
              <p className="text-sm text-gray-300">{index.name}</p>
              <p className="text-2xl font-bold text-white mt-1">
                {index.value.toLocaleString("en-US", { minimumFractionDigits: 2 })}
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

      {/* Sector Performance */}
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
              <Tooltip cursor={{ fill: "transparent" }} itemStyle={{ color: "white" }} labelStyle={{ color: "white" }}
                contentStyle={{
                  backgroundColor: "#1f2937",
                  border: "1px solid white", color: "white",
                  borderRadius: "8px"
                }}
              />
              <Bar dataKey="change" radius={[8, 8, 0, 0]} activeBar={{ stroke: "white", strokeWidth: 2 }}>
                {sectorPerformance.map((entry) => (
                  <Cell key={`cell-${entry.sector}`} fill={entry.change >= 0 ? "#10b981" : "#ef4444"} />
                ))}
              </Bar>
            </BarChart>
          </ResponsiveContainer>
        </CardContent>
      </Card>

      {/* Market Activity */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
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
                <Tooltip cursor={{ fill: "transparent" }} itemStyle={{ color: "white" }} labelStyle={{ color: "white" }}
                  contentStyle={{
                    backgroundColor: "#1f2937",
                    border: "1px solid white", color: "white",
                    borderRadius: "8px"
                  }}
                />
                <Bar dataKey="volume" fill="#3b82f6" radius={[8, 8, 0, 0]} activeBar={{ stroke: "white", strokeWidth: 2 }} />
              </BarChart>
            </ResponsiveContainer>
          </CardContent>
        </Card>

        <Card className="text-white">
          <CardHeader>
            <CardTitle className="text-white">Market Breadth</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-6">
              <div>
                <div className="flex justify-between mb-2">
                  <span className="text-sm text-gray-300">Advancing</span>
                  <span className="text-sm font-semibold text-green-600">1,845 stocks</span>
                </div>
                <div className="h-3 bg-gray-100 rounded-full overflow-hidden">
                  <div className="h-full bg-green-600 w-[65%]"></div>
                </div>
              </div>
              <div>
                <div className="flex justify-between mb-2">
                  <span className="text-sm text-gray-300">Declining</span>
                  <span className="text-sm font-semibold text-red-600">992 stocks</span>
                </div>
                <div className="h-3 bg-gray-100 rounded-full overflow-hidden">
                  <div className="h-full bg-red-600 w-[35%]"></div>
                </div>
              </div>
              <div className="pt-4 border-t space-y-3">
                <div className="flex justify-between">
                  <span className="text-gray-300">New Highs</span>
                  <span className="font-semibold text-white">234</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-300">New Lows</span>
                  <span className="font-semibold text-white">45</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-300">Unchanged</span>
                  <span className="font-semibold text-white">163</span>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Market Movers */}
      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList className="grid w-full grid-cols-3 max-w-2xl">
          <TabsTrigger value="overview">
            <TrendingUp className="w-4 h-4 mr-2" />
            Top Gainers
          </TabsTrigger>
          <TabsTrigger value="losers">
            <ArrowDownRight className="w-4 h-4 mr-2" />
            Top Losers
          </TabsTrigger>
          <TabsTrigger value="active">
            <Activity className="w-4 h-4 mr-2" />
            Most Active
          </TabsTrigger>
        </TabsList>

        <TabsContent value="overview">
          <Card className="text-white">
            <CardContent className="pt-6">
              <div className="space-y-3">
                {gainers.map((stock) => (
                  <div
                    key={stock.symbol}
                    className="flex items-center justify-between p-4 border border-transparent hover:border-white hover:bg-transparent rounded-lg cursor-pointer transition-colors"
                    onClick={() => navigate(`/stock/${stock.symbol}`)}
                  >
                    <div>
                      <p className="font-semibold text-white">{stock.symbol}</p>
                      <p className="text-sm text-gray-300">{stock.name}</p>
                    </div>
                    <div className="text-right">
                      <p className="font-semibold text-white">${stock.price.toFixed(2)}</p>
                      <div className="flex items-center gap-1 text-green-600">
                        <ArrowUpRight className="w-4 h-4" />
                        <span className="text-sm font-semibold">
                          +{stock.changePercent.toFixed(2)}%
                        </span>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="losers">
          <Card className="text-white">
            <CardContent className="pt-6">
              <div className="space-y-3">
                {losers.map((stock) => (
                  <div
                    key={stock.symbol}
                    className="flex items-center justify-between p-4 border border-transparent hover:border-white hover:bg-transparent rounded-lg cursor-pointer transition-colors"
                    onClick={() => navigate(`/stock/${stock.symbol}`)}
                  >
                    <div>
                      <p className="font-semibold text-white">{stock.symbol}</p>
                      <p className="text-sm text-gray-300">{stock.name}</p>
                    </div>
                    <div className="text-right">
                      <p className="font-semibold text-white">${stock.price.toFixed(2)}</p>
                      <div className="flex items-center gap-1 text-red-600">
                        <ArrowDownRight className="w-4 h-4" />
                        <span className="text-sm font-semibold">
                          {stock.changePercent.toFixed(2)}%
                        </span>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="active">
          <Card className="text-white">
            <CardContent className="pt-6">
              <div className="space-y-3">
                {mostActive.map((stock) => (
                  <div
                    key={stock.symbol}
                    className="flex items-center justify-between p-4 border border-transparent hover:border-white hover:bg-transparent rounded-lg cursor-pointer transition-colors"
                    onClick={() => navigate(`/stock/${stock.symbol}`)}
                  >
                    <div>
                      <p className="font-semibold text-white">{stock.symbol}</p>
                      <p className="text-sm text-gray-300">{stock.name}</p>
                    </div>
                    <div className="text-right">
                      <p className="font-semibold text-white">
                        {(stock.volume / 1000000).toFixed(2)}M
                      </p>
                      <p className="text-sm text-gray-300">Volume</p>
                    </div>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}