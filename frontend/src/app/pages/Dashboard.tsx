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

export default function Dashboard() {
  const navigate = useNavigate();
  const [watchlist, setWatchlist] = useState<any[]>([]);
  const [portfolio, setPortfolio] = useState<any[]>([]);
  const [wallet, setWallet] = useState<any>({ balance: 0 });
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

        const holdings = portRes?.holdings || [];
        setPortfolio(holdings);
        setWallet(walletRes || { balance: 0 });

        let watchItems = watchRes?.watchlist || [];

        // If user has no watchlist yet, show some suggested stocks from the DB
        if (watchItems.length === 0 && stocksRes?.stocks?.length > 0) {
          watchItems = stocksRes.stocks.slice(0, 5).map((s: any) => {
            const pseudoChange = ((s.symbol.length * 7) % 10) - 5;
            return {
              ...s,
              isSuggested: true,
              change: pseudoChange || 1.2,
              changePercent: (pseudoChange / s.price) * 100 || 0.8,
            };
          });
        } else {
          watchItems = watchItems.map((s: any) => ({
            ...s,
            change: s.change || Math.random() * 10 - 5,
            changePercent: s.changePercent || Math.random() * 5 - 2.5,
          }));
        }

        setWatchlist(watchItems);
      } catch (error) {
        console.error("Failed to load dashboard data", error);
      } finally {
        setLoading(false);
      }
    };

    fetchDashboardData();

    // Simulate real-time price updates for the UI
    const interval = setInterval(() => {
      setWatchlist((prev) =>
        prev.map((item) => {
          const changeDelta = (Math.random() - 0.5) * 2;
          const newPrice = Math.max(0.01, item.price + changeDelta);
          const change = item.change + changeDelta;
          const changePercent = (change / (item.price - item.change)) * 100;
          return {
            ...item,
            price: newPrice,
            change: parseFloat(change.toFixed(2)),
            changePercent: parseFloat(changePercent.toFixed(2)),
          };
        }),
      );
    }, 5000);

    return () => clearInterval(interval);
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
