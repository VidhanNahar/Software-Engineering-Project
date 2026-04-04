import { useState, useEffect, useMemo } from "react";
import { useNavigate } from "react-router";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "../components/ui/dialog";
import {
  ArrowUpRight,
  ArrowDownRight,
  TrendingUp,
  Plus,
  Star,
  Loader2,
  Search,
  CheckCircle2,
} from "lucide-react";
import { toast } from "sonner";
import { portfolioApi, watchlistApi, stockApi, walletApi } from "../api";

interface WatchlistItem {
  symbol: string;
  name: string;
  price: number;
  change: number;
  changePercent: number;
  watchlist_id?: string;
  stock_id?: string;
  isSuggested?: boolean;
}

interface PortfolioItem {
  symbol: string;
  quantity: number;
  currentPrice: number;
  avgPrice: number;
  totalGainLoss: number;
}

interface WalletData {
  balance: number;
}

interface StockOption {
  stock_id: string;
  symbol: string;
  name: string;
  price: number;
}

function toNumber(value: unknown): number {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : 0;
}

export default function Dashboard() {
  const navigate = useNavigate();
  const [watchlist, setWatchlist] = useState<WatchlistItem[]>([]);
  const [portfolio, setPortfolio] = useState<PortfolioItem[]>([]);
  const [wallet, setWallet] = useState<WalletData>({ balance: 0 });
  const [loading, setLoading] = useState(true);

  // Add-to-watchlist dialog state
  const [addDialogOpen, setAddDialogOpen] = useState(false);
  const [allStocks, setAllStocks] = useState<StockOption[]>([]);
  const [stockSearch, setStockSearch] = useState("");
  const [addingStockId, setAddingStockId] = useState<string | null>(null);

  const watchlistStockIds = useMemo(
    () => new Set(watchlist.map((w) => w.stock_id).filter(Boolean)),
    [watchlist],
  );

  const filteredStocks = useMemo(() => {
    const q = stockSearch.toLowerCase().trim();
    return allStocks.filter((s) => {
      const notInWatchlist = !watchlistStockIds.has(s.stock_id);
      if (!q) return notInWatchlist;
      return (
        notInWatchlist &&
        (s.symbol.toLowerCase().includes(q) || s.name.toLowerCase().includes(q))
      );
    });
  }, [allStocks, stockSearch, watchlistStockIds]);

  const openAddDialog = async () => {
    setStockSearch("");
    setAddDialogOpen(true);
    if (allStocks.length === 0) {
      try {
        const res = await stockApi.getAll();
        setAllStocks(
          (res?.stocks || []).map((s: Record<string, unknown>) => ({
            stock_id: String(s.stock_id || s.id || ""),
            symbol: String(s.symbol || ""),
            name: String(s.name || ""),
            price: toNumber(s.price),
          })),
        );
      } catch {
        toast.error("Failed to load stocks");
      }
    }
  };

  const handleAddToWatchlist = async (stock: StockOption) => {
    if (addingStockId) return;
    setAddingStockId(stock.stock_id);
    try {
      await watchlistApi.add({ stock_id: stock.stock_id });
      toast.success(`${stock.symbol} added to your watchlist!`);
      // Optimistically add to watchlist state
      setWatchlist((prev) => [
        ...prev,
        {
          symbol: stock.symbol,
          name: stock.name,
          price: stock.price,
          change: 0,
          changePercent: 0,
          stock_id: stock.stock_id,
        },
      ]);
      setAddDialogOpen(false);
    } catch (err: unknown) {
      toast.error(
        err instanceof Error ? err.message : "Failed to add to watchlist",
      );
    } finally {
      setAddingStockId(null);
    }
  };

  useEffect(() => {
    const fetchDashboardData = async () => {
      try {
        const [portRes, watchRes, walletRes, stocksRes] = await Promise.all([
          portfolioApi.get().catch(() => ({ holdings: [] })),
          watchlistApi.get().catch(() => ({ watchlist: [] })),
          walletApi.get().catch(() => ({ balance: 0 })),
          stockApi.getAll().catch(() => ({ stocks: [] })),
        ]);

        const holdings = (portRes?.holdings || []).map(
          (holding: Record<string, unknown>) => {
            const quantity = toNumber(holding.quantity);
            const currentPrice = toNumber(
              holding.currentPrice ?? holding.current_price ?? holding.price,
            );
            const avgPrice = toNumber(
              holding.avgPrice ??
                holding.average_price ??
                holding.avg_price ??
                holding.price,
            );
            const totalGainLoss =
              holding.totalGainLoss !== undefined ||
              holding.total_gain_loss !== undefined
                ? toNumber(holding.totalGainLoss ?? holding.total_gain_loss)
                : quantity * currentPrice - quantity * avgPrice;

            return {
              symbol: String(holding.symbol || "UNK"),
              quantity,
              currentPrice,
              avgPrice,
              totalGainLoss,
            } satisfies PortfolioItem;
          },
        );
        setPortfolio(holdings);
        setWallet(walletRes || { balance: 0 });

        // Cache all stocks for the add-dialog
        if (stocksRes?.stocks?.length > 0) {
          setAllStocks(
            stocksRes.stocks.map((s: Record<string, unknown>) => ({
              stock_id: String(s.stock_id || s.id || ""),
              symbol: String(s.symbol || ""),
              name: String(s.name || ""),
              price: toNumber(s.price),
            })),
          );
        }

        let watchItems = watchRes?.watchlist || [];

        // If user has no watchlist yet, show some suggested stocks from the DB
        if (watchItems.length === 0 && stocksRes?.stocks?.length > 0) {
          watchItems = stocksRes.stocks
            .slice(0, 5)
            .map((s: Record<string, unknown>) => ({
              ...s,
              isSuggested: true,
              change: s.change ?? 0,
              changePercent: s.changePercent ?? s.change_percent ?? 0,
            }));
        } else {
          watchItems = watchItems.map((s: Record<string, unknown>) => ({
            ...s,
            change: s.change ?? 0,
            changePercent: s.changePercent ?? s.change_percent ?? 0,
          }));
        }

        setWatchlist(watchItems as WatchlistItem[]);
      } catch (error) {
        console.error("Failed to load dashboard data", error);
      } finally {
        setLoading(false);
      }
    };

    fetchDashboardData();

    let ws: WebSocket | null = null;
    try {
      const wsProtocol = window.location.protocol === "https:" ? "wss" : "ws";
      ws = new WebSocket(`${wsProtocol}://${window.location.host}/ws/stocks`);

      ws.onopen = () => {
        console.log("💫 Dashboard WebSocket connected");
      };

      ws.onmessage = (evt) => {
        try {
          const data = JSON.parse(evt.data);
          if (data.market_open === false) return;

          if (data.type === "stock_tick" || data.type === "stocks_snapshot") {
            const incomingStocks = data.stocks || data.ticks || [];

            setWatchlist((prev) =>
              prev.map((item) => {
                const updated = incomingStocks.find(
                  (s: Record<string, unknown>) => s.symbol === item.symbol,
                );
                if (updated && typeof updated.price === "number") {
                  const newPrice = updated.price;
                  const oldPrice = item.price;
                  const change = newPrice - oldPrice;
                  const changePercent =
                    oldPrice !== 0 ? (change / oldPrice) * 100 : 0;
                  return { ...item, price: newPrice, change, changePercent };
                }
                return item;
              }),
            );

            setPortfolio((prev) =>
              prev.map((item) => {
                const updated = incomingStocks.find(
                  (s: Record<string, unknown>) => s.symbol === item.symbol,
                );
                if (updated && typeof updated.price === "number") {
                  const newPrice = updated.price;
                  return {
                    ...item,
                    currentPrice: newPrice,
                    totalGainLoss:
                      item.quantity * newPrice - item.quantity * item.avgPrice,
                  };
                }
                return item;
              }),
            );
          }
        } catch {
          // Ignore malformed payloads
        }
      };

      ws.onerror = () => console.log("⚠️ Dashboard WebSocket error");
      ws.onclose = () => console.log("🔌 Dashboard WebSocket disconnected");
    } catch (error) {
      console.error("Failed to connect to WebSocket", error);
    }

    return () => {
      if (ws) ws.close();
    };
  }, []);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
      </div>
    );
  }

  const totalValue = portfolio.reduce(
    (sum, item) => sum + item.quantity * item.currentPrice,
    0,
  );
  const totalInvested = portfolio.reduce(
    (sum, item) => sum + item.quantity * item.avgPrice,
    0,
  );
  const totalGainLoss = totalValue - totalInvested;
  const returnPercent =
    totalInvested > 0 ? (totalGainLoss / totalInvested) * 100 : 0;

  return (
    <div className="space-y-6">
      {/* ── Add to Watchlist Dialog ── */}
      <Dialog open={addDialogOpen} onOpenChange={setAddDialogOpen}>
        <DialogContent className="max-w-md bg-card border-border">
          <DialogHeader>
            <DialogTitle className="text-lg">
              Add Stock to Watchlist
            </DialogTitle>
          </DialogHeader>

          <div className="relative mt-2">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
            <Input
              autoFocus
              placeholder="Search by symbol or name…"
              value={stockSearch}
              onChange={(e) => setStockSearch(e.target.value)}
              className="pl-9 bg-input-background border-border"
            />
          </div>

          <div className="mt-3 max-h-72 overflow-y-auto space-y-1 pr-1">
            {allStocks.length === 0 ? (
              <div className="flex items-center justify-center py-8">
                <Loader2 className="w-5 h-5 animate-spin text-primary mr-2" />
                <span className="text-muted-foreground text-sm">Loading stocks…</span>
              </div>
            ) : filteredStocks.length === 0 ? (
              <p className="text-center text-muted-foreground text-sm py-8">
                {stockSearch
                  ? `No results for "${stockSearch}"`
                  : "All available stocks are already in your watchlist."}
              </p>
            ) : (
              filteredStocks.map((stock) => (
                <button
                  key={stock.stock_id}
                  disabled={addingStockId === stock.stock_id}
                  onClick={() => handleAddToWatchlist(stock)}
                  className="w-full flex items-center justify-between px-3 py-2.5 rounded-lg hover:bg-accent transition-colors disabled:opacity-60 text-left group"
                >
                  <div>
                    <p className="font-semibold text-sm">
                      {stock.symbol}
                    </p>
                    <p className="text-xs text-muted-foreground truncate max-w-[220px]">
                      {stock.name}
                    </p>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="text-sm">
                      ${stock.price.toFixed(2)}
                    </span>
                    {addingStockId === stock.stock_id ? (
                      <Loader2 className="w-4 h-4 animate-spin text-primary" />
                    ) : (
                      <CheckCircle2 className="w-4 h-4 text-primary opacity-0 group-hover:opacity-100 transition-opacity" />
                    )}
                  </div>
                </button>
              ))
            )}
          </div>
        </DialogContent>
      </Dialog>

      {/* ── Page Header ── */}
      <div>
        <h1 className="text-3xl font-bold">Dashboard</h1>
        <p className="text-muted-foreground mt-1">
          Welcome back. Here's your trading overview.
        </p>
      </div>

      {/* ── Summary Cards ── */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card>
          <CardContent className="p-6">
            <p className="text-sm text-muted-foreground">Wallet Balance</p>
            <p className="text-2xl font-bold mt-1">
              ₹
              {wallet.balance.toLocaleString("en-US", {
                minimumFractionDigits: 2,
              })}
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-6">
            <p className="text-sm text-muted-foreground">Total Portfolio Value</p>
            <p className="text-2xl font-bold mt-1">
              ₹
              {totalValue.toLocaleString("en-US", {
                minimumFractionDigits: 2,
              })}
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-6">
            <p className="text-sm text-muted-foreground">Today's Return</p>
            <p
              className={`text-2xl font-bold mt-1 ${
                totalGainLoss >= 0 ? "text-green-500" : "text-red-500"
              }`}
            >
              {totalGainLoss >= 0 ? "+" : ""}₹
              {totalGainLoss.toLocaleString("en-US", {
                minimumFractionDigits: 2,
              })}
              <span className="text-sm ml-2">
                ({returnPercent >= 0 ? "+" : ""}
                {returnPercent.toFixed(2)}%)
              </span>
            </p>
          </CardContent>
        </Card>
      </div>

      {/* ── Portfolio + Watchlist ── */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Portfolio */}
        <div className="lg:col-span-2 space-y-6">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between">
              <div>
                <CardTitle>Your Portfolio</CardTitle>
                <p className="text-sm text-muted-foreground mt-1">
                  Your investment performance
                </p>
              </div>
              <Button
                variant="outline"
                onClick={() => navigate("/portfolio")}
              >
                View All
              </Button>
            </CardHeader>
            <CardContent>
              {portfolio.length === 0 ? (
                <div className="text-center py-8">
                  <TrendingUp className="w-12 h-12 text-muted-foreground mx-auto mb-3" />
                  <p className="text-muted-foreground">Your portfolio is empty.</p>
                  <Button
                    className="mt-4"
                    onClick={() => navigate("/market")}
                  >
                    Explore Market
                  </Button>
                </div>
              ) : (
                <div className="space-y-4">
                  {portfolio.slice(0, 3).map((holding, index) => (
                    <div
                      key={index}
                      className="flex items-center justify-between p-4 border border-transparent hover:border-border hover:bg-accent/50 rounded-lg transition-colors cursor-pointer"
                      onClick={() => navigate(`/stock/${holding.symbol}`)}
                    >
                      <div>
                        <p className="font-semibold">
                          {holding.symbol}
                        </p>
                        <p className="text-sm text-muted-foreground">
                          {holding.quantity} Shares
                        </p>
                      </div>
                      <div className="text-right">
                        <p className="font-semibold">
                          ₹
                          {(
                            holding.quantity * holding.currentPrice
                          ).toLocaleString("en-US", {
                            minimumFractionDigits: 2,
                          })}
                        </p>
                        <p
                          className={`text-sm ${
                            holding.totalGainLoss >= 0
                              ? "text-green-500"
                              : "text-red-500"
                          }`}
                        >
                          {holding.totalGainLoss >= 0 ? "+" : ""}₹
                          {holding.totalGainLoss.toFixed(2)}
                        </p>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </div>

        {/* Watchlist */}
        <div>
          <Card>
            <CardHeader className="flex flex-row items-center justify-between">
              <CardTitle>Watchlist</CardTitle>
              <Button
                variant="ghost"
                size="icon"
                title="Add stock to watchlist"
                onClick={openAddDialog}
              >
                <Plus className="w-5 h-5" />
              </Button>
            </CardHeader>
            <CardContent>
              {watchlist.length === 0 ? (
                <div className="text-center py-8">
                  <Star className="w-10 h-10 text-muted-foreground mx-auto mb-2" />
                  <p className="text-muted-foreground text-sm">
                    No stocks in your watchlist yet.
                  </p>
                  <Button
                    variant="ghost"
                    className="mt-2 text-primary hover:text-primary/80 text-sm"
                    onClick={openAddDialog}
                  >
                    Add your first stock
                  </Button>
                </div>
              ) : (
                <div className="space-y-4">
                  {watchlist.map((stock) => (
                    <div
                      key={stock.symbol}
                      className="flex items-center justify-between p-3 border border-transparent hover:border-border hover:bg-accent/50 rounded-lg cursor-pointer transition-colors"
                      onClick={() => navigate(`/stock/${stock.symbol}`)}
                    >
                      <div className="flex items-center gap-3">
                        <Star
                          className={`w-4 h-4 flex-shrink-0 ${
                            stock.isSuggested
                              ? "text-muted-foreground"
                              : "text-yellow-500 fill-yellow-500"
                          }`}
                          onClick={async (e) => {
                            e.stopPropagation();
                            if (stock.isSuggested) {
                              const sId = stock.stock_id;
                              if (sId) {
                                try {
                                  await watchlistApi.add({ stock_id: sId });
                                  toast.success(
                                    `${stock.symbol} added to watchlist!`,
                                  );
                                  setWatchlist((prev) =>
                                    prev.map((w) =>
                                      w.symbol === stock.symbol
                                        ? { ...w, isSuggested: false }
                                        : w,
                                    ),
                                  );
                                } catch (err: unknown) {
                                  toast.error(
                                    err instanceof Error
                                      ? err.message
                                      : "Failed to add to watchlist",
                                  );
                                }
                              }
                            }
                          }}
                        />
                        <div>
                          <p className="font-semibold">
                            {stock.symbol}
                          </p>
                          <p className="text-xs text-muted-foreground">
                            {stock.isSuggested ? "Suggested" : stock.name}
                          </p>
                        </div>
                      </div>
                      <div className="text-right">
                        <p className="font-semibold">
                          ${stock.price?.toFixed(2) ?? "—"}
                        </p>
                        <div
                          className={`flex items-center gap-1 text-xs ${
                            stock.change >= 0
                              ? "text-green-500"
                              : "text-red-500"
                          }`}
                        >
                          {stock.change >= 0 ? (
                            <ArrowUpRight className="w-3 h-3" />
                          ) : (
                            <ArrowDownRight className="w-3 h-3" />
                          )}
                          <span>
                            {Math.abs(stock.changePercent).toFixed(2)}%
                          </span>
                        </div>
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
