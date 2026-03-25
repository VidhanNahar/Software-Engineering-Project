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
  Plus,
  Star,
  Loader2,
} from "lucide-react";
import { portfolioApi, watchlistApi, stockApi, walletApi } from "../api";

interface WatchlistItem {
  symbol: string;
  name: string;
  price: number;
  change: number;
  changePercent: number;
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

  useEffect(() => {
    const fetchDashboardData = async () => {
      try {
        // Fetch real data from backend, catch errors to allow partial loads
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
              holding.avgPrice ?? holding.average_price ?? holding.avg_price ?? holding.price,
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

        let watchItems = watchRes?.watchlist || [];

        // If user has no watchlist yet, show some suggested stocks from the DB
        if (watchItems.length === 0 && stocksRes?.stocks?.length > 0) {
          watchItems = stocksRes.stocks
            .slice(0, 5)
            .map((s: Record<string, unknown>) => {
              return {
                ...s,
                isSuggested: true,
                change: s.change ?? 0,
                changePercent: s.changePercent ?? s.change_percent ?? 0,
              } as WatchlistItem;
            });
        } else {
          watchItems = watchItems.map(
            (s: Record<string, unknown>) =>
              ({
                ...s,
                change: s.change ?? 0,
                changePercent: s.changePercent ?? s.change_percent ?? 0,
              }) as WatchlistItem,
          );
        }

        setWatchlist(watchItems);
      } catch (error) {
        console.error("Failed to load dashboard data", error);
      } finally {
        setLoading(false);
      }
    };

    fetchDashboardData();

    // Establish WebSocket connection for real-time updates
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
          
          // If market is closed, don't update prices
          if (data.market_open === false) {
            console.debug("📊 Market closed, freezing Dashboard prices");
            return;
          }
          
          // Handle real-time stock updates
          if (data.type === "stock_tick" || data.type === "stocks_snapshot") {
            const incomingStocks = data.stocks || data.ticks || [];
            
            // Update watchlist with new prices
            setWatchlist((prev) =>
              prev.map((item) => {
                const updated = incomingStocks.find(
                  (s: Record<string, unknown>) => s.symbol === item.symbol
                );
                if (updated && typeof updated.price === "number") {
                  const newPrice = updated.price;
                  const oldPrice = item.price;
                  const change = newPrice - oldPrice;
                  const changePercent = oldPrice !== 0 ? (change / oldPrice) * 100 : 0;
                  
                  return {
                    ...item,
                    price: newPrice,
                    change,
                    changePercent,
                  };
                }
                return item;
              })
            );

            // Update portfolio with new prices
            setPortfolio((prev) =>
              prev.map((item) => {
                const updated = incomingStocks.find(
                  (s: Record<string, unknown>) => s.symbol === item.symbol
                );
                if (updated && typeof updated.price === "number") {
                  const newPrice = updated.price;
                  const newTotalGainLoss =
                    item.quantity * newPrice -
                    item.quantity * item.avgPrice;
                  return {
                    ...item,
                    currentPrice: newPrice,
                    totalGainLoss: newTotalGainLoss,
                  };
                }
                return item;
              })
            );
          }
        } catch (e) {
          // Ignore malformed WebSocket payloads
        }
      };

      ws.onerror = () => {
        console.log("⚠️ Dashboard WebSocket error");
      };

      ws.onclose = () => {
        console.log("🔌 Dashboard WebSocket disconnected");
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
      <div className="flex items-center justify-center h-full">
        <Loader2 className="w-8 h-8 animate-spin text-blue-500" />
      </div>
    );
  }

  // Calculate portfolio totals
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
    <div className="space-y-6 text-white">
      <div>
        <h1 className="text-3xl font-bold text-white">Dashboard</h1>
        <p className="text-gray-300 mt-1">
          Welcome back. Here's your trading overview.
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card className="text-white">
          <CardContent className="p-6">
            <p className="text-sm text-gray-300">Wallet Balance</p>
            <p className="text-2xl font-bold text-white mt-1">
              $
              {wallet.balance.toLocaleString("en-US", {
                minimumFractionDigits: 2,
              })}
            </p>
          </CardContent>
        </Card>
        <Card className="text-white">
          <CardContent className="p-6">
            <p className="text-sm text-gray-300">Total Portfolio Value</p>
            <p className="text-2xl font-bold text-white mt-1">
              $
              {totalValue.toLocaleString("en-US", { minimumFractionDigits: 2 })}
            </p>
          </CardContent>
        </Card>
        <Card className="text-white">
          <CardContent className="p-6">
            <p className="text-sm text-gray-300">Today's Return</p>
            <p
              className={`text-2xl font-bold mt-1 ${totalGainLoss >= 0 ? "text-green-500" : "text-red-500"}`}
            >
              {totalGainLoss >= 0 ? "+" : ""}$
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

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="lg:col-span-2 space-y-6">
          <Card className="text-white">
            <CardHeader className="flex flex-row items-center justify-between">
              <div>
                <CardTitle className="text-white">Your Portfolio</CardTitle>
                <p className="text-sm text-gray-300 mt-1">
                  Your investment performance
                </p>
              </div>
              <Button
                variant="outline"
                className="text-gray-900 dark:text-white"
                onClick={() => navigate("/portfolio")}
              >
                View All
              </Button>
            </CardHeader>
            <CardContent>
              {portfolio.length === 0 ? (
                <div className="text-center py-8">
                  <TrendingUp className="w-12 h-12 text-gray-600 mx-auto mb-3" />
                  <p className="text-gray-300">Your portfolio is empty.</p>
                  <Button
                    className="mt-4 bg-blue-600 hover:bg-blue-700 text-white"
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
                      className="flex items-center justify-between p-4 border border-transparent hover:border-white hover:bg-transparent rounded-lg transition-colors cursor-pointer"
                      onClick={() => navigate(`/stock/${holding.symbol}`)}
                    >
                      <div>
                        <p className="font-semibold text-white">
                          {holding.symbol}
                        </p>
                        <p className="text-sm text-gray-300">
                          {holding.quantity} Shares
                        </p>
                      </div>
                      <div className="text-right">
                        <p className="font-semibold text-white">
                          $
                          {(
                            holding.quantity * holding.currentPrice
                          ).toLocaleString("en-US", {
                            minimumFractionDigits: 2,
                          })}
                        </p>
                        <p
                          className={`text-sm ${holding.totalGainLoss >= 0 ? "text-green-500" : "text-red-500"}`}
                        >
                          {holding.totalGainLoss >= 0 ? "+" : ""}$
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

        <div>
          <Card className="text-white">
            <CardHeader className="flex flex-row items-center justify-between">
              <CardTitle className="text-white">Watchlist</CardTitle>
              <Button
                variant="ghost"
                size="icon"
                className="text-white hover:bg-gray-800"
              >
                <Plus className="w-5 h-5" />
              </Button>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {watchlist.map((stock) => (
                  <div
                    key={stock.symbol}
                    className="flex items-center justify-between p-3 border border-transparent hover:border-white hover:bg-transparent rounded-lg cursor-pointer transition-colors"
                    onClick={() => navigate(`/stock/${stock.symbol}`)}
                  >
                    <div className="flex items-center gap-3">
                      <Star
                        className={`w-4 h-4 ${stock.isSuggested ? "text-gray-500" : "text-yellow-500 fill-yellow-500"}`}
                      />
                      <div>
                        <p className="font-semibold text-white">
                          {stock.symbol}
                        </p>
                        <p className="text-xs text-gray-300">
                          {stock.isSuggested ? "Suggested" : stock.name}
                        </p>
                      </div>
                    </div>
                    <div className="text-right">
                      <p className="font-semibold text-white">
                        ${stock.price?.toFixed(2)}
                      </p>
                      <div
                        className={`flex items-center gap-1 text-xs ${
                          stock.change >= 0 ? "text-green-500" : "text-red-500"
                        }`}
                      >
                        {stock.change >= 0 ? (
                          <ArrowUpRight className="w-3 h-3" />
                        ) : (
                          <ArrowDownRight className="w-3 h-3" />
                        )}
                        <span>{Math.abs(stock.changePercent).toFixed(2)}%</span>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
