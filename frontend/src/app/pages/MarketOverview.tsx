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
} from "recharts";
import { stockApi } from "../api";

type MarketStock = {
  stock_id: string;
  symbol: string;
  name: string;
  price: number;
  change: number;
  change_percent: number;
  volume: number;
  total_traded_value: number;
};

function toNumber(value: unknown): number {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : 0;
}

function normalizeStocks(
  rawStocks: {
    symbol: string;
    name: string;
    price: number;
    quantity: number;
    volume?: number;
    change?: number;
    changePercent?: number;
    change_percent?: number;
  }[] = [],
): MarketStock[] {
  return rawStocks.map((s) => ({
    stock_id: String(s.stock_id || ""),
    symbol: String(s.symbol || "").toUpperCase(),
    name: String(s.name || ""),
    price: toNumber(s.price),
    change: toNumber(s.change),
    change_percent: toNumber(s.change_percent),
    volume: toNumber(s.volume),
    total_traded_value: toNumber(s.total_traded_value),
  }));
}

export default function MarketOverview() {
  const navigate = useNavigate();
  const [activeTab, setActiveTab] = useState("gainers");
  const [allStocks, setAllStocks] = useState<MarketStock[]>([]);
  const [loading, setLoading] = useState(true);
  const [wsConnected, setWsConnected] = useState(false);

  useEffect(() => {
    const fetchStocks = async () => {
      try {
        const response = await stockApi.getAll();
        if (response?.stocks) {
          setAllStocks(normalizeStocks(response.stocks));
        }
      } catch (err) {
        console.error("Failed to fetch stocks", err);
      } finally {
        setLoading(false);
      }
    };
    fetchStocks();
  }, []);

  useEffect(() => {
    let ws: WebSocket | null = null;
    const wsProtocol = window.location.protocol === "https:" ? "wss" : "ws";

    try {
      ws = new WebSocket(`${wsProtocol}://${window.location.host}/ws/stocks`);

      ws.onopen = () => setWsConnected(true);
      ws.onclose = () => setWsConnected(false);
      ws.onerror = () => setWsConnected(false);

      ws.onmessage = (event: MessageEvent<string>) => {
        try {
          const payload = JSON.parse(event.data) as {
            stocks?: {
              symbol: string;
              name: string;
              price: number;
              quantity: number;
              volume?: number;
              change?: number;
              changePercent?: number;
            }[];
          };
          if (!payload?.stocks) return;
          setAllStocks(normalizeStocks(payload.stocks));
        } catch {
          // Ignore malformed websocket payloads.
        }
      };
    } catch {
      setWsConnected(false);
    }

    return () => {
      if (ws) ws.close();
    };
  }, []);

  const gainers = [...allStocks]
    .sort((a, b) => b.change_percent - a.change_percent)
    .slice(0, 5);
  const losers = [...allStocks]
    .sort((a, b) => a.change_percent - b.change_percent)
    .slice(0, 5);
  const mostActive = [...allStocks]
    .sort((a, b) => b.volume - a.volume)
    .slice(0, 5);

  const topByTradedValue = [...allStocks]
    .sort((a, b) => b.total_traded_value - a.total_traded_value)
    .slice(0, 4);

  const moversData = [...allStocks]
    .sort((a, b) => Math.abs(b.change_percent) - Math.abs(a.change_percent))
    .slice(0, 8)
    .map((s) => ({ symbol: s.symbol, change_percent: s.change_percent }));

  const volumeData = [...allStocks]
    .sort((a, b) => b.volume - a.volume)
    .slice(0, 8)
    .map((s) => ({ symbol: s.symbol, volume: s.volume }));

  const advancing = allStocks.filter((s) => s.change > 0).length;
  const declining = allStocks.filter((s) => s.change < 0).length;
  const unchanged = allStocks.length - advancing - declining;
  const advancingPct =
    allStocks.length > 0 ? (advancing / allStocks.length) * 100 : 0;
  const decliningPct =
    allStocks.length > 0 ? (declining / allStocks.length) * 100 : 0;

  if (loading) {
    return <div className="p-6">Loading market data...</div>;
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold">Market Overview</h1>
        <p className="text-muted-foreground mt-1">
          Live market data from backend stream
        </p>
        <p
          className={`mt-1 text-sm ${wsConnected ? "text-green-500" : "text-yellow-500"}`}
        >
          Feed:{" "}
          {wsConnected
            ? "Live websocket connected"
            : "Using latest API snapshot"}
        </p>
      </div>

      {/* Market Leaders */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {topByTradedValue.map((stock) => (
          <Card key={stock.symbol}>
            <CardContent className="p-6">
              <p className="text-sm text-muted-foreground">{stock.symbol}</p>
              <p className="text-xs text-muted-foreground/80 truncate">{stock.name}</p>
              <p className="text-2xl font-bold mt-1">
                ₹
                {stock.price.toLocaleString("en-US", {
                  minimumFractionDigits: 2,
                  maximumFractionDigits: 2,
                })}
              </p>
              <div
                className={`flex items-center gap-1 mt-1 ${
                  stock.change >= 0 ? "text-green-600" : "text-red-600"
                }`}
              >
                {stock.change >= 0 ? (
                  <ArrowUpRight className="w-4 h-4" />
                ) : (
                  <ArrowDownRight className="w-4 h-4" />
                )}
                <span className="text-sm font-semibold">
                  {stock.change >= 0 ? "+" : ""}
                  {stock.change.toFixed(2)} (
                  {stock.change_percent >= 0 ? "+" : ""}
                  {stock.change_percent.toFixed(2)}%)
                </span>
              </div>
              <p className="mt-1 text-xs text-muted-foreground">
                Traded Value: {stock.total_traded_value.toLocaleString("en-US")}
              </p>
            </CardContent>
          </Card>
        ))}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Main Charts */}
        <div className="lg:col-span-2 space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Top Movers (%)</CardTitle>
            </CardHeader>
            <CardContent>
              <ResponsiveContainer width="100%" height={300}>
                <BarChart data={moversData}>
                  <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
                  <XAxis dataKey="symbol" stroke="var(--muted-foreground)" />
                  <YAxis stroke="var(--muted-foreground)" />
                  <Tooltip
                    cursor={{ fill: "transparent" }}
                    itemStyle={{ color: "var(--foreground)" }}
                    labelStyle={{ color: "var(--foreground)" }}
                    contentStyle={{
                      backgroundColor: "var(--popover)",
                      border: "1px solid var(--border)",
                      color: "var(--foreground)",
                      borderRadius: "8px",
                    }}
                  />
                  <Bar
                    dataKey="change_percent"
                    radius={[8, 8, 0, 0]}
                    activeBar={{ stroke: "var(--foreground)", strokeWidth: 2 }}
                    fill="var(--primary)"
                  ></Bar>
                </BarChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>

          {/* Market Activity */}
          <Card>
            <CardHeader>
              <CardTitle>Most Active Volume</CardTitle>
            </CardHeader>
            <CardContent>
              <ResponsiveContainer width="100%" height={250}>
                <BarChart data={volumeData}>
                  <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
                  <XAxis dataKey="symbol" stroke="var(--muted-foreground)" />
                  <YAxis stroke="var(--muted-foreground)" />
                  <Tooltip
                    cursor={{ fill: "transparent" }}
                    itemStyle={{ color: "var(--foreground)" }}
                    labelStyle={{ color: "var(--foreground)" }}
                    contentStyle={{
                      backgroundColor: "var(--popover)",
                      border: "1px solid var(--border)",
                      color: "var(--foreground)",
                      borderRadius: "8px",
                    }}
                  />
                  <Bar
                    dataKey="volume"
                    fill="var(--primary)"
                    radius={[8, 8, 0, 0]}
                    activeBar={{ stroke: "var(--foreground)", strokeWidth: 2 }}
                  />
                </BarChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Market Breadth</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm text-muted-foreground">
                  Advancing ({advancingPct.toFixed(0)}%)
                </span>
                <span className="text-sm text-muted-foreground">
                  Declining ({decliningPct.toFixed(0)}%)
                </span>
              </div>
              <div className="w-full h-4 flex rounded-full overflow-hidden">
                <div
                  className="bg-green-500 h-full"
                  style={{ width: `${advancingPct}%` }}
                />
                <div
                  className="bg-red-500 h-full"
                  style={{ width: `${decliningPct}%` }}
                />
              </div>
              <div className="flex justify-between mt-4">
                <div className="text-center">
                  <p className="text-2xl font-bold text-green-500">
                    {advancing}
                  </p>
                  <p className="text-sm text-muted-foreground">Issues Advancing</p>
                </div>
                <div className="text-center">
                  <p className="text-2xl font-bold text-red-500">{declining}</p>
                  <p className="text-sm text-muted-foreground">Issues Declining</p>
                </div>
                <div className="text-center">
                  <p className="text-2xl font-bold text-muted-foreground">
                    {unchanged}
                  </p>
                  <p className="text-sm text-muted-foreground">Unchanged</p>
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
              <TabsTrigger value="gainers">Gainers</TabsTrigger>
              <TabsTrigger value="losers">Losers</TabsTrigger>
              <TabsTrigger value="active">Active</TabsTrigger>
            </TabsList>

            <TabsContent value="gainers">
              <Card>
                <CardContent className="pt-6">
                  <div className="space-y-4">
                    {gainers.map((stock) => (
                      <div
                        key={stock.symbol}
                        className="flex items-center justify-between p-4 border border-transparent hover:border-border hover:bg-accent/50 rounded-lg cursor-pointer transition-colors"
                        onClick={() => navigate(`/stock/${stock.symbol}`)}
                      >
                        <div>
                          <p className="font-semibold">
                            {stock.symbol}
                          </p>
                          <p className="text-sm text-muted-foreground">{stock.name}</p>
                        </div>
                        <div className="text-right">
                          <p className="font-semibold">
                            ${stock.price.toFixed(2)}
                          </p>
                          <p className="text-sm text-green-500 flex items-center justify-end gap-1">
                            <ArrowUpRight className="w-4 h-4" />
                            {stock.change_percent.toFixed(2)}%
                          </p>
                        </div>
                      </div>
                    ))}
                  </div>
                </CardContent>
              </Card>
            </TabsContent>

            <TabsContent value="losers">
              <Card>
                <CardContent className="pt-6">
                  <div className="space-y-4">
                    {losers.map((stock) => (
                      <div
                        key={stock.symbol}
                        className="flex items-center justify-between p-4 border border-transparent hover:border-border hover:bg-accent/50 rounded-lg cursor-pointer transition-colors"
                        onClick={() => navigate(`/stock/${stock.symbol}`)}
                      >
                        <div>
                          <p className="font-semibold">
                            {stock.symbol}
                          </p>
                          <p className="text-sm text-muted-foreground">{stock.name}</p>
                        </div>
                        <div className="text-right">
                          <p className="font-semibold">
                            ${stock.price.toFixed(2)}
                          </p>
                          <p className="text-sm text-red-500 flex items-center justify-end gap-1">
                            <ArrowDownRight className="w-4 h-4" />
                            {Math.abs(stock.change_percent).toFixed(2)}%
                          </p>
                        </div>
                      </div>
                    ))}
                  </div>
                </CardContent>
              </Card>
            </TabsContent>

            <TabsContent value="active">
              <Card>
                <CardContent className="pt-6">
                  <div className="space-y-4">
                    {mostActive.map((stock) => (
                      <div
                        key={stock.symbol}
                        className="flex items-center justify-between p-4 border border-transparent hover:border-border hover:bg-accent/50 rounded-lg cursor-pointer transition-colors"
                        onClick={() => navigate(`/stock/${stock.symbol}`)}
                      >
                        <div>
                          <p className="font-semibold">
                            {stock.symbol}
                          </p>
                          <p className="text-sm text-muted-foreground">{stock.name}</p>
                        </div>
                        <div className="text-right">
                          <p className="font-semibold">
                            ${stock.price.toFixed(2)}
                          </p>
                          <p className="text-sm text-muted-foreground">
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
