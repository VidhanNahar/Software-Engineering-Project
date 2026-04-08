import { useState, useEffect, useRef } from "react";
import { useNavigate } from "react-router";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../components/ui/card";
import { Button } from "../components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../components/ui/select";
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
import { portfolioApi, transactionApi, walletApi, stockApi, adminApi, ordersApi, WS_STOCKS_URL } from "../api";
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
  stock_id?: string;
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
  locked_balance?: number;
}

interface PendingOrderItem {
  order_id: string;
  stock_id: string;
  order_type: string;
  limit_price: number;
  quantity: number;
  filled_quantity: number;
  status: string;
  created_at?: string;
}

const normalizePendingOrder = (order: any): PendingOrderItem => ({
  order_id: order.order_id ?? order.OrderID,
  stock_id: order.stock_id ?? order.StockID,
  order_type: order.order_type ?? order.OrderType,
  limit_price: Number(order.limit_price ?? order.LimitPrice ?? 0),
  quantity: Number(order.quantity ?? order.Quantity ?? 0),
  filled_quantity: Number(order.filled_quantity ?? order.FilledQuantity ?? 0),
  status: (order.status ?? order.Status ?? "").toString(),
  created_at: order.created_at ?? order.CreatedAt,
});

const normalizeStatus = (status?: string) =>
  (status || "").toString().trim().toUpperCase();

const isActiveOrder = (order: PendingOrderItem) => {
  const s = normalizeStatus(order.status);
  return s === "PENDING" || s === "PARTIALLY_FILLED" || s === "PENDING_APPROVAL";
};

export default function Portfolio() {
  const navigate = useNavigate();
  const [holdings, setHoldings] = useState<HoldingItem[]>([]);
  const [trades, setTrades] = useState<TradeHistoryItem[]>([]);
  const [wallet, setWallet] = useState<WalletData>({ balance: 0, locked_balance: 0 });
  const [loading, setLoading] = useState(true);
  const [pendingOrders, setPendingOrders] = useState<PendingOrderItem[]>([]);
  const [stockLookup, setStockLookup] = useState<Record<string, { symbol: string; name: string; price?: number }>>({});
  const [activityFilter, setActivityFilter] = useState<"all" | "pending" | "market">("all");
  const [cancelingOrderId, setCancelingOrderId] = useState<string | null>(null);
  const [isMarketOpen, setIsMarketOpen] = useState(true);
  const isMarketOpenRef = useRef(true);

  useEffect(() => {
    const fetchPortfolioData = async () => {
      try {
        const [portRes, transRes, walletRes, stocksRes, pendingRes] = await Promise.all([
          portfolioApi.get().catch(() => ({ holdings: [] })),
          transactionApi.getHistory().catch(() => ({ history: [] })),
          walletApi.get().catch(() => ({ balance: 0 })),
          stockApi.getAll().catch(() => ({ stocks: [] })),
          ordersApi.getPendingOrders().catch(() => ({ pending_orders: [] })),
        ]);

        const stockMap = new Map();
        if (stocksRes.stocks) {
          const stockEntries: Record<string, { symbol: string; name: string; price?: number }> = {};
          stocksRes.stocks.forEach(
            (s: {
              stock_id: string;
              price?: number;
              name?: string;
              symbol?: string;
            }) => {
              stockMap.set(s.stock_id, s);
              stockEntries[s.stock_id] = {
                symbol: s.symbol || "STK",
                name: s.name || "Unknown Stock",
                price: s.price,
              };
            },
          );
          setStockLookup(stockEntries);
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
            avg_buy_price?: number;
            current_price?: number;
            currency_code?: string;
            currencyCode?: string;
          }) => {
            const stockInfo = stockMap.get(h.stock_id);
            const currentPrice = h.current_price || 0;
            const avgPrice = h.avg_buy_price || currentPrice;
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
        setWallet({
          balance: Number(walletRes?.balance ?? 0),
          locked_balance: Number(walletRes?.locked_balance ?? 0),
        });
        const rawLimitOrders =
          pendingRes?.limit_orders || pendingRes?.pending_orders || pendingRes?.orders || [];
        setPendingOrders(rawLimitOrders.map(normalizePendingOrder));
      } catch (error) {
        toast.error("Failed to load portfolio data");
      } finally {
        setLoading(false);
      }
    };

    fetchPortfolioData();

    const refreshInterval = setInterval(fetchPortfolioData, 15000);

    // Fetch initial market status
    const fetchMarketStatus = async () => {
      try {
        const status = await adminApi.getMarketStatus();
        if (status?.is_open !== undefined) {
          setIsMarketOpen(status.is_open);
          isMarketOpenRef.current = status.is_open;
          console.log(
            status.is_open
              ? "✅ Market OPEN on page load"
              : "🔒 Market CLOSED on page load"
          );
        }
      } catch (err) {
        console.log("Failed to fetch market status on load", err);
        // Default to true if fetch fails
        setIsMarketOpen(true);
        isMarketOpenRef.current = true;
      }
    };

    fetchMarketStatus();

    // Establish WebSocket connection for real-time price updates
    let ws: WebSocket | null = null;
    try {
      ws = new WebSocket(WS_STOCKS_URL);

      ws.onopen = () => {
        console.log("💫 Portfolio WebSocket connected");
      };

      ws.onmessage = (evt) => {
        try {
          const data = JSON.parse(evt.data);

          // Handle market status updates
          if (data.type === "market_status") {
            setIsMarketOpen(data.market_open);
            isMarketOpenRef.current = data.market_open;  // Update ref immediately
            if (data.market_open === false) {
              console.log("🔒 MARKET CLOSED - All prices frozen");
            } else {
              console.log("🔓 MARKET OPENED - Live prices resuming");
            }
            return;
          }

          // Check if market is open using ref (synchronous check)
          if (!isMarketOpenRef.current) {
            console.debug("📊 Market closed, freezing Portfolio prices");
            return;
          }

          // Update holdings with real-time price changes
          if (data.type === "stock_tick" || data.type === "stocks_snapshot") {
            const incomingStocks = data.stocks || data.ticks || [];

            setStockLookup((prev) => {
              const next = { ...prev };
              for (const tick of incomingStocks) {
                if (!tick || typeof tick.symbol !== "string" || typeof tick.price !== "number") {
                  continue;
                }
                const key = Object.keys(next).find(
                  (stockId) => next[stockId]?.symbol === tick.symbol,
                );
                if (key) {
                  next[key] = {
                    ...next[key],
                    price: tick.price,
                  };
                }
              }
              return next;
            });

            setHoldings((prev) =>
              prev.map((holding) => {
                const updated = incomingStocks.find(
                  (s: Record<string, unknown>) => s.symbol === holding.symbol,
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
                        ? (newTotalGainLoss /
                            (holding.quantity * holding.avgPrice)) *
                          100
                        : 0,
                  };
                }
                return holding;
              }),
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
      clearInterval(refreshInterval);
      if (ws) {
        ws.close();
      }
    };
  }, []);

  const handleCancelPendingOrder = async (orderId: string) => {
    try {
      setCancelingOrderId(orderId);
      await ordersApi.cancelPendingOrder(orderId);
      toast.success("Pending order cancelled");
      const pendingRes = await ordersApi.getPendingOrders().catch(() => ({ pending_orders: [] }));
      const rawLimitOrders = pendingRes?.limit_orders || pendingRes?.pending_orders || pendingRes?.orders || [];
      setPendingOrders(rawLimitOrders.map(normalizePendingOrder));
      const walletRes = await walletApi.get().catch(() => ({ balance: 0, locked_balance: 0 }));
      setWallet({
        balance: Number(walletRes?.balance ?? 0),
        locked_balance: Number(walletRes?.locked_balance ?? 0),
      });
    } catch (error: any) {
      toast.error(error?.message || "Failed to cancel pending order");
    } finally {
      setCancelingOrderId(null);
    }
  };

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

  const getStockLabel = (stockId?: string, fallbackSymbol?: string) => {
    if (fallbackSymbol) return fallbackSymbol;
    return stockLookup[stockId || ""]?.symbol || holdings.find((h: any) => h.stock_id === stockId)?.symbol || "STK";
  };

  const getStockName = (stockId?: string) =>
    stockLookup[stockId || ""]?.name || "Unknown Stock";

  const getStockCurrentPrice = (stockId?: string) =>
    stockLookup[stockId || ""]?.price ?? 0;

  const visiblePendingOrders = pendingOrders.filter(isActiveOrder);

  const visibleMarketTrades = trades.filter((t: any) => {
    const mode = (t.order_type || t.mode || "").toString().toUpperCase();
    if (mode) return mode === "MARKET";
    return true;
  });

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
      {!isMarketOpen && (
        <div className="bg-red-900/20 border border-red-500/50 rounded-lg p-4 text-center">
          <p className="text-red-400 font-semibold text-lg">🔒 Market Closed</p>
          <p className="text-red-300 text-sm mt-1">All prices are frozen. Live updates will resume when market opens.</p>
        </div>
      )}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold">Portfolio</h1>
          <p className="text-muted-foreground mt-1">
            Track your investments and performance
          </p>
        </div>
        <div className="flex gap-2">
          <div className="w-[220px]">
            <Select
              value={activityFilter}
              onValueChange={(v: "all" | "pending" | "market") => setActivityFilter(v)}
            >
              <SelectTrigger>
                <Filter className="w-4 h-4 mr-2" />
                <SelectValue placeholder="Filter" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Activity</SelectItem>
                <SelectItem value="pending">List Orders</SelectItem>
                <SelectItem value="market">Market Orders</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <Button variant="outline">
            <Download className="w-4 h-4 mr-2" />
            Export
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardContent className="p-6">
            <p className="text-sm text-muted-foreground">Total Value</p>
            <p className="text-2xl font-bold mt-1">{formatPrice(totalValue)}</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-6">
            <p className="text-sm text-muted-foreground">Buying Power</p>
            <p className="text-2xl font-bold mt-1">
              {formatPrice(wallet.balance)}
            </p>
            <p className="text-xs text-muted-foreground mt-1">
              Locked: {formatPrice(wallet.locked_balance || 0)}
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
                  <p className="text-muted-foreground">
                    You don't own any stocks yet.
                  </p>
                  <Button className="mt-4" onClick={() => navigate("/market")}>
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
                          <p className="font-medium">{holding.quantity}</p>
                        </div>
                        <div>
                          <p className="text-muted-foreground mb-1">
                            Avg Price
                          </p>
                          <p className="font-medium">
                            {formatPrice(holding.avgPrice)}
                          </p>
                        </div>
                        <div>
                          <p className="text-muted-foreground mb-1">
                            Current Price
                          </p>
                          <p className="font-medium">
                            {formatPrice(holding.currentPrice)}
                          </p>
                        </div>
                        <div>
                          <p className="text-muted-foreground mb-1">
                            Portfolio %
                          </p>
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
                          rawProps: unknown,
                        ) => {
                          const props = rawProps as {
                            payload?: { percent?: number };
                          };
                          const percent = props?.payload?.percent || 0;
                          return [
                            `${formatPrice(value)} (${percent.toFixed(1)}%)`,
                            name,
                          ];
                        }}
                      />
                    </PieChart>
                  </ResponsiveContainer>
                </div>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>
                {activityFilter === "pending"
                  ? "List Orders"
                  : activityFilter === "market"
                    ? "Market Orders"
                    : "Recent Trades"}
              </CardTitle>
            </CardHeader>
            <CardContent>
              {activityFilter === "pending" ? (
                visiblePendingOrders.length === 0 ? (
                  <p className="text-muted-foreground text-center py-4">
                    No list orders
                  </p>
                ) : (
                  <div className="space-y-4">
                    {visiblePendingOrders.map((order) => (
                      <div
                        key={order.order_id}
                        className="flex items-center justify-between p-3 border border-border rounded-lg"
                      >
                        <div>
                          <div className="flex items-center gap-2">
                            <span
                              className={`px-2 py-0.5 rounded text-xs font-semibold uppercase ${
                                order.order_type === "BUY"
                                  ? "bg-green-500/20 text-green-500"
                                  : "bg-red-500/20 text-red-500"
                              }`}
                            >
                              {order.order_type}
                            </span>
                            <span className="font-bold">
                              {getStockLabel(order.stock_id)}
                            </span>
                          </div>
                          <p className="text-xs text-muted-foreground mt-1">
                            {getStockName(order.stock_id)}
                          </p>
                          <p className="text-xs text-muted-foreground mt-1">
                            Current: {formatPrice(getStockCurrentPrice(order.stock_id))} • Limit: {formatPrice(Number(order.limit_price || 0))} • Qty: {order.quantity - order.filled_quantity}
                          </p>
                        </div>
                        <div className="flex flex-col items-end gap-2">
                          <span className="text-xs text-muted-foreground">
                            {normalizeStatus(order.status) || "UNKNOWN"}
                          </span>
                          {isActiveOrder(order) ? (
                            <Button
                              size="sm"
                              variant="outline"
                              onClick={() => handleCancelPendingOrder(order.order_id)}
                              disabled={cancelingOrderId === order.order_id || !isMarketOpen}
                            >
                              {!isMarketOpen
                                ? "Market Closed"
                                : cancelingOrderId === order.order_id
                                  ? "Cancelling..."
                                  : "Cancel"}
                            </Button>
                          ) : (
                            <Button size="sm" variant="outline" disabled>
                              Not cancellable
                            </Button>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>
                )
              ) : (activityFilter === "market" ? visibleMarketTrades : trades).length === 0 ? (
                <p className="text-muted-foreground text-center py-4">
                  No activity found for this filter
                </p>
              ) : (
                <div className="space-y-4">
                  {(activityFilter === "market" ? visibleMarketTrades : trades).slice(0, 8).map((trade, i) => (
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
                            {trade.type || trade.transaction_type || "EXECUTED"}
                          </span>
                          <span className="font-bold">
                            {trade.symbol || getStockLabel(trade.stock_id, trade.symbol)}
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
                          {trade.quantity} @{" "}
                          {formatPrice(
                            Number(trade.price || trade.price_per_stock || 0),
                          )}
                        </p>
                        <p className="text-sm text-muted-foreground">
                          {formatPrice(
                            trade.quantity *
                              Number(trade.price || trade.price_per_stock || 0),
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
