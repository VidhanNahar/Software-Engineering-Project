import { useState, useEffect } from "react";
import { useNavigate } from "react-router";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../components/ui/card";
import { Button } from "../components/ui/button";
import {
  ArrowUpRight,
  ArrowDownRight,
  TrendingUp,
  Download,
  Filter,
  Loader2,
} from "lucide-react";
import { PieChart, Pie, Cell, ResponsiveContainer, Tooltip } from "recharts";
import { formatPrice } from "../utils/currency";
import { portfolioApi, transactionApi, walletApi, stockApi } from "../api";
import { toast } from "sonner";

interface HoldingItem {
  symbol: string;
  name?: string;
  quantity: number;
  avgPrice: number;
  currentPrice: number;
  totalInvested: number;
  totalValue: number;
  totalGainLoss: number;
  returnPercent?: number;
  gainLossPercent: number;
}

interface TradeHistoryItem {
  transaction_id?: string;
  symbol: string;
  type?: string;
  transaction_type?: string;
  quantity: number;
  price: number;
  price_per_stock?: number;
  timestamp?: string;
  status?: string;
}

interface WalletData {
  balance: number;
}

export default function Portfolio() {
  const navigate = useNavigate();
  const [holdings, setHoldings] = useState<HoldingItem[]>([]);
  const [trades, setTrades] = useState<TradeHistoryItem[]>([]);
  const [wallet, setWallet] = useState<WalletData>({ balance: 0 });
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchPortfolioData = async () => {
      try {
        const [portRes, transRes, walletRes, stocksRes] = await Promise.all([
          portfolioApi.get().catch(() => ({ holdings: [] })),
          transactionApi.getHistory().catch(() => ({ history: [] })),
          walletApi.get().catch(() => ({ balance: 0 })),
          stockApi.getAll().catch(() => ({ stocks: [] })),
        ]);

        const stockMap = new Map();
        if (stocksRes.stocks) {
          stocksRes.stocks.forEach(
            (s: {
              stock_id: string;
              price?: number;
              name?: string;
              symbol?: string;
            }) => {
              stockMap.set(s.stock_id, s);
            },
          );
        }

        let fetchedHoldings = portRes?.holdings || [];
        if (fetchedHoldings.length === 0 && portRes?.portfolio) {
          fetchedHoldings = portRes.portfolio;
        }

        // Enrich holdings with current prices
        const enrichedHoldings = fetchedHoldings.map(
          (h: {
            stock_id: string;
            symbol?: string;
            quantity?: number;
            average_price?: number;
            price?: number;
            currency_code?: string;
            currencyCode?: string;
          }) => {
            const stockInfo = stockMap.get(h.stock_id);
            const currentPrice =
              stockInfo?.price || h.average_price || h.price || 0;
            const avgPrice = h.average_price || h.price || currentPrice;
            const quantity = h.quantity || 0;

            const totalValue = quantity * currentPrice;
            const totalInvested = quantity * avgPrice;
            const totalGainLoss = totalValue - totalInvested;

            return {
              ...h,
              symbol: h.symbol || stockInfo?.symbol || "UNK",
              name: stockInfo?.name || "Unknown Stock",
              quantity,
              avgPrice,
              currentPrice,
              totalValue,
              totalGainLoss,
              gainLossPercent:
                totalInvested > 0 ? (totalGainLoss / totalInvested) * 100 : 0,
            };
          },
        );

        setHoldings(enrichedHoldings);

        const enrichedTrades = (
          transRes?.history ||
          transRes?.transactions ||
          []
        ).map((t: any) => {
          return {
            ...t,
          };
        });
        setTrades(enrichedTrades);
        setWallet(walletRes || { balance: 0 });
      } catch (error) {
        console.error("Failed to load portfolio", error);
        toast.error("Failed to load portfolio data");
      } finally {
        setLoading(false);
      }
    };

    fetchPortfolioData();

    // Establish WebSocket connection for real-time price updates
    let ws: WebSocket | null = null;
    try {
      const wsProtocol = window.location.protocol === "https:" ? "wss" : "ws";
      ws = new WebSocket(`${wsProtocol}://${window.location.host}/ws/stocks`);

      ws.onopen = () => {
        console.log("💫 Portfolio WebSocket connected");
      };

      ws.onmessage = (evt) => {
        try {
          const data = JSON.parse(evt.data);

          // If market is closed, don't update prices
          if (data.market_open === false) {
            console.debug("📊 Market closed, freezing Portfolio prices");
            return;
          }

          // Update holdings with real-time price changes
          if (data.type === "stock_tick" || data.type === "stocks_snapshot") {
            const incomingStocks = data.stocks || data.ticks || [];

            setHoldings((prev) =>
              prev.map((holding) => {
                const updated = incomingStocks.find(
                  (s: Record<string, unknown>) => s.symbol === holding.symbol
                );
                if (updated && typeof updated.price === "number") {
                  const newPrice = updated.price;
                  const newTotalValue = holding.quantity * newPrice;
                  const newTotalGainLoss =
                    newTotalValue - holding.quantity * holding.avgPrice;

                  return {
                    ...holding,
                    currentPrice: newPrice,
                    totalValue: newTotalValue,
                    totalGainLoss: newTotalGainLoss,
                    gainLossPercent:
                      holding.quantity * holding.avgPrice > 0
                        ? (newTotalGainLoss / (holding.quantity * holding.avgPrice)) *
                          100
                        : 0,
                  };
                }
                return holding;
              })
            );
          }
        } catch (e) {
          // Ignore malformed WebSocket payloads
        }
      };

      ws.onerror = () => {
        console.log("⚠️ Portfolio WebSocket error");
      };

      ws.onclose = () => {
        console.log("🔌 Portfolio WebSocket disconnected");
      };
    } catch (error) {
      console.error("Failed to connect to WebSocket", error);
    }

    return () => {
      if (ws) {
        ws.close();
      }
    };
  }, []);

  if (loading) {
    return (
      <div className="flex justify-center items-center h-full">
        <Loader2 className="w-10 h-10 animate-spin text-primary" />
      </div>
    );
  }

  const totalValue = holdings.reduce((sum, h) => sum + h.totalValue, 0);
  const totalGainLoss = holdings.reduce((sum, h) => sum + h.totalGainLoss, 0);
  const totalInvested = totalValue - totalGainLoss;
  const overallReturn =
    totalInvested > 0 ? (totalGainLoss / totalInvested) * 100 : 0;

  const portfolioDistribution = holdings
    .map((holding) => ({
      name: holding.symbol,
      value: holding.totalValue,
      percent: totalValue > 0 ? (holding.totalValue / totalValue) * 100 : 0,
    }))
    .filter((h) => h.value > 0);

  const COLORS = [
    "#3b82f6",
    "#10b981",
    "#f59e0b",
    "#ef4444",
    "#8b5cf6",
    "#ec4899",
    "#14b8a6",
  ];

  return (
    <div className="space-y-6 max-w-7xl mx-auto">
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold">Portfolio</h1>
          <p className="text-muted-foreground mt-1">
            Track your investments and performance
          </p>
        </div>
        <div className="flex gap-2">
          <Button
            variant="outline"
          >
            <Filter className="w-4 h-4 mr-2" />
            Filter
          </Button>
          <Button
            variant="outline"
          >
            <Download className="w-4 h-4 mr-2" />
            Export
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardContent className="p-6">
            <p className="text-sm text-muted-foreground">Total Value</p>
            <p className="text-2xl font-bold mt-1">
              {formatPrice(totalValue)}
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-6">
            <p className="text-sm text-muted-foreground">Buying Power</p>
            <p className="text-2xl font-bold mt-1">
              {formatPrice(wallet.balance)}
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-6">
            <p className="text-sm text-muted-foreground">Total Gain/Loss</p>
            <div className="flex items-center gap-2 mt-1">
              <p
                className={`text-2xl font-bold ${
                  totalGainLoss >= 0 ? "text-green-500" : "text-red-500"
                }`}
              >
                {totalGainLoss >= 0 ? "+" : ""}
                {formatPrice(Math.abs(totalGainLoss))}
              </p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-6">
            <p className="text-sm text-muted-foreground">Overall Return</p>
            <div className="flex items-center gap-2 mt-1">
              <p
                className={`text-2xl font-bold ${
                  overallReturn >= 0 ? "text-green-500" : "text-red-500"
                }`}
              >
                {overallReturn >= 0 ? "+" : ""}
                {overallReturn.toFixed(2)}%
              </p>
            </div>
          </CardContent>
        </Card>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="lg:col-span-2 space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Holdings</CardTitle>
            </CardHeader>
            <CardContent>
              {holdings.length === 0 ? (
                <div className="text-center py-12">
                  <TrendingUp className="w-12 h-12 text-muted-foreground mx-auto mb-3" />
                  <p className="text-muted-foreground">You don't own any stocks yet.</p>
                  <Button
                    className="mt-4"
                    onClick={() => navigate("/market")}
                  >
                    Start Trading
                  </Button>
                </div>
              ) : (
                <div className="space-y-4">
                  {holdings.map((holding) => (
                    <div
                      key={holding.symbol}
                      className="p-4 border border-border rounded-lg hover:border-primary/50 hover:bg-accent/50 cursor-pointer transition-colors"
                      onClick={() => navigate(`/stock/${holding.symbol}`)}
                    >
                      <div className="flex items-start justify-between mb-3">
                        <div>
                          <p className="font-semibold text-lg">
                            {holding.symbol}
                          </p>
                          <p className="text-sm text-muted-foreground">
                            {holding.name}
                          </p>
                        </div>
                        <div className="text-right">
                          <p className="font-semibold">
                            {formatPrice(holding.totalValue)}
                          </p>
                          <div
                            className={`text-sm font-medium ${
                              holding.totalGainLoss >= 0
                                ? "text-green-500"
                                : "text-red-500"
                            }`}
                          >
                            {holding.totalGainLoss >= 0 ? "+" : ""}
                            {formatPrice(Math.abs(holding.totalGainLoss))} (
                            {holding.gainLossPercent >= 0 ? "+" : ""}
                            {holding.gainLossPercent.toFixed(2)}%)
                          </div>
                        </div>
                      </div>
                      <div className="grid grid-cols-4 gap-4 text-sm mt-4 pt-4 border-t border-border">
                        <div>
                          <p className="text-muted-foreground mb-1">Quantity</p>
                          <p className="font-medium">
                            {holding.quantity}
                          </p>
                        </div>
                        <div>
                          <p className="text-muted-foreground mb-1">Avg Price</p>
                          <p className="font-medium">
                            {formatPrice(holding.avgPrice)}
                          </p>
                        </div>
                        <div>
                          <p className="text-muted-foreground mb-1">Current Price</p>
                          <p className="font-medium">
                            {formatPrice(holding.currentPrice)}
                          </p>
                        </div>
                        <div>
                          <p className="text-muted-foreground mb-1">Portfolio %</p>
                          <p className="font-medium">
                            {(
                              (holding.totalValue / totalValue) * 100 || 0
                            ).toFixed(1)}
                            %
                          </p>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </div>

        <div className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Asset Allocation</CardTitle>
            </CardHeader>
            <CardContent>
              {holdings.length === 0 ? (
                <p className="text-center py-8 text-muted-foreground">
                  No assets to display
                </p>
              ) : (
                <div className="h-[250px] w-full">
                  <ResponsiveContainer width="100%" height="100%">
                    <PieChart>
                      <Pie
                        data={portfolioDistribution}
                        cx="50%"
                        cy="50%"
                        innerRadius={60}
                        outerRadius={80}
                        paddingAngle={5}
                        dataKey="value"
                      >
                        {portfolioDistribution.map((entry, index) => (
                          <Cell
                            key={`cell-${index}`}
                            fill={COLORS[index % COLORS.length]}
                            stroke="transparent"
                          />
                        ))}
                      </Pie>
                      <Tooltip
                        itemStyle={{ color: "var(--foreground)" }}
                        labelStyle={{ color: "var(--foreground)" }}
                        contentStyle={{
                          backgroundColor: "var(--popover)",
                          border: "1px solid var(--border)",
                          color: "var(--foreground)",
                          borderRadius: "8px",
                        }}
                        formatter={(
                          value: number,
                          name: string,
                          props: { payload: { percent: number } },
                        ) => [
                          `${formatPrice(value)} (${props.payload.percent.toFixed(1)}%)`,
                          name,
                        ]}
                      />
                    </PieChart>
                  </ResponsiveContainer>
                </div>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Recent Trades</CardTitle>
            </CardHeader>
            <CardContent>
              {trades.length === 0 ? (
                <p className="text-muted-foreground text-center py-4">
                  No recent activity
                </p>
              ) : (
                <div className="space-y-4">
                  {trades.slice(0, 5).map((trade, i) => (
                    <div
                      key={i}
                      className="flex items-center justify-between p-3 border border-border rounded-lg"
                    >
                      <div>
                        <div className="flex items-center gap-2">
                          <span
                            className={`px-2 py-0.5 rounded text-xs font-semibold uppercase ${
                              trade.type === "BUY" || trade.type === "buy"
                                ? "bg-green-500/20 text-green-500"
                                : "bg-red-500/20 text-red-500"
                            }`}
                          >
                            {trade.type}
                          </span>
                          <span className="font-bold">
                            {trade.symbol || "STK"}
                          </span>
                        </div>
                        <p className="text-xs text-muted-foreground mt-1">
                          {new Date(
                            trade.timestamp || Date.now(),
                          ).toLocaleDateString()}
                        </p>
                      </div>
                      <div className="text-right">
                        <p className="font-semibold">
                          {trade.quantity} @ {formatPrice(Number(trade.price || trade.price_per_stock || 0))}
                        </p>
                        <p className="text-sm text-muted-foreground">
                          {formatPrice(
                            trade.quantity *
                            Number(trade.price || trade.price_per_stock || 0)
                          )}
                        </p>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
