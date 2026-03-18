import { useState, useEffect } from "react";
import { useNavigate } from "react-router";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { ArrowUpRight, ArrowDownRight, TrendingUp, Plus, Star } from "lucide-react";
import { defaultWatchlist, defaultPortfolio, simulatePriceUpdate } from "../utils/mockData";
import { WatchlistItem, PortfolioHolding } from "../types/stock";

export default function Dashboard() {
  const navigate = useNavigate();
  const [watchlist, setWatchlist] = useState<WatchlistItem[]>(defaultWatchlist);
  const [portfolio] = useState<PortfolioHolding[]>(defaultPortfolio);

  // Simulate real-time price updates
  useEffect(() => {
    const interval = setInterval(() => {
      setWatchlist((prev) =>
        prev.map((item) => {
          const newPrice = simulatePriceUpdate(item.price);
          const change = newPrice - item.price;
          const changePercent = (change / item.price) * 100;
          return {
            ...item,
            price: newPrice,
            change: parseFloat(change.toFixed(2)),
            changePercent: parseFloat(changePercent.toFixed(2)),
          };
        })
      );
    }, 3000);

    return () => clearInterval(interval);
  }, []);

  const totalPortfolioValue = portfolio.reduce((sum, holding) => sum + holding.totalValue, 0);
  const totalGainLoss = portfolio.reduce((sum, holding) => sum + holding.totalGainLoss, 0);
  const portfolioGainLossPercent = (totalGainLoss / (totalPortfolioValue - totalGainLoss)) * 100;

  const marketStats = [
    { name: "S&P 500", value: "5,234.18", change: "+0.87%", isPositive: true },
    { name: "Dow Jones", value: "38,905.66", change: "+1.24%", isPositive: true },
    { name: "NASDAQ", value: "16,315.70", change: "-0.23%", isPositive: false },
  ];

  return (
    <div className="space-y-6">
      {/* Welcome Header */}
      <div>
        <h1 className="text-3xl font-bold text-foreground">Dashboard</h1>
        <p className="text-muted-foreground mt-1">Welcome back, John. Here's your trading overview.</p>
      </div>

      {/* Market Indices */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {marketStats.map((stat) => (
          <Card key={stat.name}>
            <CardContent className="p-6">
              <p className="text-sm text-muted-foreground">{stat.name}</p>
              <p className="text-2xl font-bold text-foreground mt-1">{stat.value}</p>
              <p
                className={`text-sm mt-1 ${
                  stat.isPositive ? "text-green-600" : "text-red-600"
                }`}
              >
                {stat.change}
              </p>
            </CardContent>
          </Card>
        ))}
      </div>

      {/* Portfolio Summary */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <div>
            <CardTitle>Portfolio Summary</CardTitle>
            <p className="text-sm text-muted-foreground mt-1">Your investment performance</p>
          </div>
          <Button onClick={() => navigate("/portfolio")}>View Details</Button>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            <div>
              <p className="text-sm text-muted-foreground">Total Value</p>
              <p className="text-3xl font-bold text-foreground mt-1">
                ${totalPortfolioValue.toLocaleString("en-US", { minimumFractionDigits: 2 })}
              </p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Total Gain/Loss</p>
              <div className="flex items-center gap-2 mt-1">
                <p
                  className={`text-3xl font-bold ${
                    totalGainLoss >= 0 ? "text-green-600" : "text-red-600"
                  }`}
                >
                  {totalGainLoss >= 0 ? "+" : ""}$
                  {totalGainLoss.toLocaleString("en-US", { minimumFractionDigits: 2 })}
                </p>
                {totalGainLoss >= 0 ? (
                  <ArrowUpRight className="w-6 h-6 text-green-600" />
                ) : (
                  <ArrowDownRight className="w-6 h-6 text-red-600" />
                )}
              </div>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Return</p>
              <p
                className={`text-3xl font-bold mt-1 ${
                  portfolioGainLossPercent >= 0 ? "text-green-600" : "text-red-600"
                }`}
              >
                {portfolioGainLossPercent >= 0 ? "+" : ""}
                {portfolioGainLossPercent.toFixed(2)}%
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Watchlist */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <div>
              <CardTitle>Watchlist</CardTitle>
              <p className="text-sm text-muted-foreground mt-1">Track your favorite stocks</p>
            </div>
            <Button variant="outline" size="sm">
              <Plus className="w-4 h-4 mr-2" />
              Add Stock
            </Button>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {watchlist.map((stock) => (
                <div
                  key={stock.symbol}
                  className="flex items-center justify-between p-3 hover:bg-accent rounded-lg cursor-pointer transition-colors"
                  onClick={() => navigate(`/stock/${stock.symbol}`)}
                >
                  <div className="flex items-center gap-3">
                    <Star className="w-4 h-4 text-yellow-500 fill-yellow-500" />
                    <div>
                      <p className="font-semibold text-foreground">{stock.symbol}</p>
                      <p className="text-sm text-muted-foreground">{stock.name}</p>
                    </div>
                  </div>
                  <div className="text-right">
                    <p className="font-semibold text-foreground">
                      ${stock.price.toFixed(2)}
                    </p>
                    <div
                      className={`flex items-center gap-1 text-sm ${
                        stock.change >= 0 ? "text-green-600" : "text-red-600"
                      }`}
                    >
                      {stock.change >= 0 ? (
                        <ArrowUpRight className="w-4 h-4" />
                      ) : (
                        <ArrowDownRight className="w-4 h-4" />
                      )}
                      <span>
                        {stock.change >= 0 ? "+" : ""}
                        {stock.changePercent.toFixed(2)}%
                      </span>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Top Movers */}
        <Card>
          <CardHeader>
            <CardTitle>Top Movers</CardTitle>
            <p className="text-sm text-muted-foreground mt-1">Biggest market movements today</p>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {[
                { symbol: "NVDA", name: "NVIDIA Corp", change: 12.34, percent: 1.42 },
                { symbol: "META", name: "Meta Platforms", change: 7.89, percent: 1.64 },
                { symbol: "MSFT", name: "Microsoft", change: 5.67, percent: 1.39 },
                { symbol: "TSLA", name: "Tesla Inc", change: -3.45, percent: -1.79 },
                { symbol: "GOOGL", name: "Alphabet Inc", change: -1.23, percent: -0.85 },
              ].map((stock) => (
                <div
                  key={stock.symbol}
                  className="flex items-center justify-between p-3 hover:bg-accent rounded-lg cursor-pointer transition-colors"
                  onClick={() => navigate(`/stock/${stock.symbol}`)}
                >
                  <div className="flex items-center gap-3">
                    <TrendingUp
                      className={`w-4 h-4 ${
                        stock.change >= 0 ? "text-green-600" : "text-red-600"
                      }`}
                    />
                    <div>
                      <p className="font-semibold text-foreground">{stock.symbol}</p>
                      <p className="text-sm text-muted-foreground">{stock.name}</p>
                    </div>
                  </div>
                  <div className="text-right">
                    <p
                      className={`font-semibold ${
                        stock.change >= 0 ? "text-green-600" : "text-red-600"
                      }`}
                    >
                      {stock.change >= 0 ? "+" : ""}${stock.change.toFixed(2)}
                    </p>
                    <p
                      className={`text-sm ${
                        stock.change >= 0 ? "text-green-600" : "text-red-600"
                      }`}
                    >
                      {stock.change >= 0 ? "+" : ""}
                      {stock.percent.toFixed(2)}%
                    </p>
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Quick Actions */}
      <Card>
        <CardHeader>
          <CardTitle>Quick Actions</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <Button variant="outline" onClick={() => navigate("/trade")}>
              Place Order
            </Button>
            <Button variant="outline" onClick={() => navigate("/market")}>
              Market Overview
            </Button>
            <Button variant="outline" onClick={() => navigate("/portfolio")}>
              View Portfolio
            </Button>
            <Button variant="outline">
              Research Tools
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}