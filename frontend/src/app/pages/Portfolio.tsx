import { useState } from "react";
import { useNavigate } from "react-router";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import {
  ArrowUpRight,
  ArrowDownRight,
  TrendingUp,
  Download,
  Filter,
} from "lucide-react";
import { PieChart, Pie, Cell, ResponsiveContainer, Legend, Tooltip } from "recharts";
import { defaultPortfolio, recentTrades } from "../utils/mockData";

export default function Portfolio() {
  const navigate = useNavigate();
  const [portfolio] = useState(defaultPortfolio);
  const [trades] = useState(recentTrades);

  const totalValue = portfolio.reduce((sum, holding) => sum + holding.totalValue, 0);
  const totalGainLoss = portfolio.reduce((sum, holding) => sum + holding.totalGainLoss, 0);
  const totalInvested = totalValue - totalGainLoss;
  const overallReturn = (totalGainLoss / totalInvested) * 100;

  const portfolioDistribution = portfolio.map((holding) => ({
    name: holding.symbol,
    value: holding.totalValue,
    percent: (holding.totalValue / totalValue) * 100,
  }));

  const COLORS = ["#3b82f6", "#10b981", "#f59e0b", "#ef4444", "#8b5cf6"];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900">Portfolio</h1>
          <p className="text-gray-600 mt-1">Track your investments and performance</p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline">
            <Filter className="w-4 h-4 mr-2" />
            Filter
          </Button>
          <Button variant="outline">
            <Download className="w-4 h-4 mr-2" />
            Export
          </Button>
        </div>
      </div>

      {/* Portfolio Summary */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardContent className="p-6">
            <p className="text-sm text-gray-600">Total Value</p>
            <p className="text-2xl font-bold text-gray-900 mt-1">
              ${totalValue.toLocaleString("en-US", { minimumFractionDigits: 2 })}
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-6">
            <p className="text-sm text-gray-600">Total Invested</p>
            <p className="text-2xl font-bold text-gray-900 mt-1">
              ${totalInvested.toLocaleString("en-US", { minimumFractionDigits: 2 })}
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-6">
            <p className="text-sm text-gray-600">Total Gain/Loss</p>
            <div className="flex items-center gap-2 mt-1">
              <p
                className={`text-2xl font-bold ${
                  totalGainLoss >= 0 ? "text-green-600" : "text-red-600"
                }`}
              >
                {totalGainLoss >= 0 ? "+" : ""}$
                {totalGainLoss.toLocaleString("en-US", { minimumFractionDigits: 2 })}
              </p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-6">
            <p className="text-sm text-gray-600">Overall Return</p>
            <div className="flex items-center gap-2 mt-1">
              <p
                className={`text-2xl font-bold ${
                  overallReturn >= 0 ? "text-green-600" : "text-red-600"
                }`}
              >
                {overallReturn >= 0 ? "+" : ""}
                {overallReturn.toFixed(2)}%
              </p>
              {overallReturn >= 0 ? (
                <ArrowUpRight className="w-5 h-5 text-green-600" />
              ) : (
                <ArrowDownRight className="w-5 h-5 text-red-600" />
              )}
            </div>
          </CardContent>
        </Card>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Holdings Table */}
        <div className="lg:col-span-2">
          <Card>
            <CardHeader>
              <CardTitle>Holdings</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {portfolio.map((holding) => (
                  <div
                    key={holding.symbol}
                    className="p-4 border border-gray-200 rounded-lg hover:bg-gray-50 cursor-pointer transition-colors"
                    onClick={() => navigate(`/stock/${holding.symbol}`)}
                  >
                    <div className="flex items-start justify-between mb-3">
                      <div>
                        <p className="font-semibold text-gray-900 text-lg">
                          {holding.symbol}
                        </p>
                        <p className="text-sm text-gray-600">{holding.name}</p>
                      </div>
                      <div className="text-right">
                        <p className="font-semibold text-gray-900">
                          ${holding.totalValue.toLocaleString("en-US", { minimumFractionDigits: 2 })}
                        </p>
                        <div
                          className={`text-sm ${
                            holding.totalGainLoss >= 0 ? "text-green-600" : "text-red-600"
                          }`}
                        >
                          {holding.totalGainLoss >= 0 ? "+" : ""}$
                          {holding.totalGainLoss.toFixed(2)} (
                          {holding.gainLossPercent >= 0 ? "+" : ""}
                          {holding.gainLossPercent.toFixed(2)}%)
                        </div>
                      </div>
                    </div>
                    <div className="grid grid-cols-4 gap-4 text-sm">
                      <div>
                        <p className="text-gray-600">Quantity</p>
                        <p className="font-medium text-gray-900">{holding.quantity}</p>
                      </div>
                      <div>
                        <p className="text-gray-600">Avg. Price</p>
                        <p className="font-medium text-gray-900">
                          ${holding.avgPrice.toFixed(2)}
                        </p>
                      </div>
                      <div>
                        <p className="text-gray-600">Current Price</p>
                        <p className="font-medium text-gray-900">
                          ${holding.currentPrice.toFixed(2)}
                        </p>
                      </div>
                      <div>
                        <p className="text-gray-600">Portfolio %</p>
                        <p className="font-medium text-gray-900">
                          {((holding.totalValue / totalValue) * 100).toFixed(1)}%
                        </p>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
              <Button className="w-full mt-4" onClick={() => navigate("/trade")}>
                <TrendingUp className="w-4 h-4 mr-2" />
                Add New Position
              </Button>
            </CardContent>
          </Card>
        </div>

        {/* Portfolio Distribution */}
        <div className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Asset Allocation</CardTitle>
            </CardHeader>
            <CardContent>
              <ResponsiveContainer width="100%" height={250}>
                <PieChart>
                  <Pie
                    data={portfolioDistribution}
                    cx="50%"
                    cy="50%"
                    labelLine={false}
                    label={({ name, percent }) => `${name} ${percent.toFixed(0)}%`}
                    outerRadius={80}
                    fill="#8884d8"
                    dataKey="value"
                  >
                    {portfolioDistribution.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                    ))}
                  </Pie>
                  <Tooltip
                    formatter={(value: number) =>
                      `$${value.toLocaleString("en-US", { minimumFractionDigits: 2 })}`
                    }
                  />
                </PieChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Performance</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <div className="flex justify-between items-center">
                <span className="text-gray-600">Today</span>
                <span className="text-green-600 font-semibold">+$234.56</span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-gray-600">This Week</span>
                <span className="text-green-600 font-semibold">+$892.34</span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-gray-600">This Month</span>
                <span className="text-green-600 font-semibold">+$2,456.78</span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-gray-600">All Time</span>
                <span className="text-green-600 font-semibold">
                  +${totalGainLoss.toFixed(2)}
                </span>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>

      {/* Recent Trades */}
      <Card>
        <CardHeader>
          <CardTitle>Recent Trades</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            {trades.map((trade) => (
              <div
                key={trade.id}
                className="flex items-center justify-between p-3 border border-gray-200 rounded-lg"
              >
                <div className="flex items-center gap-4">
                  <div
                    className={`w-10 h-10 rounded-full flex items-center justify-center ${
                      trade.type === "buy" ? "bg-green-100" : "bg-red-100"
                    }`}
                  >
                    {trade.type === "buy" ? (
                      <ArrowUpRight className="w-5 h-5 text-green-600" />
                    ) : (
                      <ArrowDownRight className="w-5 h-5 text-red-600" />
                    )}
                  </div>
                  <div>
                    <p className="font-semibold text-gray-900">
                      {trade.type.toUpperCase()} {trade.symbol}
                    </p>
                    <p className="text-sm text-gray-600">
                      {trade.quantity} shares @ ${trade.price.toFixed(2)}
                    </p>
                  </div>
                </div>
                <div className="text-right">
                  <p className="font-semibold text-gray-900">
                    ${trade.total.toFixed(2)}
                  </p>
                  <p className="text-sm text-gray-600">
                    {trade.timestamp.toLocaleString("en-US", {
                      month: "short",
                      day: "numeric",
                      hour: "2-digit",
                      minute: "2-digit",
                    })}
                  </p>
                </div>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
