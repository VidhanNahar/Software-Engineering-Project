import { useState, useEffect } from "react";
import { useNavigate } from "react-router";
import { isAdmin } from "../utils/auth";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { toast } from "sonner";
import { formatPrice } from "../utils/currency";
import { adminApi } from "../api";
import {
  Trash2,
  Edit2,
  Loader2,
  PowerCircle,
  Play,
  Square,
  ShieldX,
} from "lucide-react";

interface Stock {
  stock_id: string;
  symbol: string;
  name: string;
  price: number;
  quantity: number;
}

interface MarketStatus {
  is_open: boolean;
  opened_at?: string;
  closed_at?: string;
}

export default function Admin() {
  const navigate = useNavigate();

  // ── All hooks must be declared unconditionally before any early return ──
  const [stocks, setStocks] = useState<Stock[]>([]);
  const [loading, setLoading] = useState(true);
  const [marketStatus, setMarketStatus] = useState<MarketStatus | null>(null);
  const [marketLoading, setMarketLoading] = useState(false);
  const [formData, setFormData] = useState({
    symbol: "",
    name: "",
    price: "",
    quantity: "",
  });
  const [editingId, setEditingId] = useState<string | null>(null);

  const fetchStocks = async () => {
    try {
      setLoading(true);
      const res = await adminApi.getTopStocks();
      setStocks(res?.stocks || []);
    } catch {
      toast.error("Failed to load stocks");
    } finally {
      setLoading(false);
    }
  };

  const fetchMarketStatus = async () => {
    try {
      const status = await adminApi.getMarketStatus();
      setMarketStatus(status);
    } catch (err: unknown) {
      console.log(
        "Failed to fetch market status",
        err instanceof Error ? err.message : err,
      );
    }
  };

  useEffect(() => {
    // Redirect non-admins away
    if (!isAdmin()) {
      navigate("/", { replace: true });
      return;
    }
    fetchStocks();
    fetchMarketStatus();
    const interval = setInterval(fetchMarketStatus, 2000);
    return () => clearInterval(interval);
  }, [navigate]);

  // ── Early return for non-admins (rendered while redirect is in flight) ──
  if (!isAdmin()) {
    return (
      <div className="flex flex-col items-center justify-center h-full gap-4 text-center">
        <div className="w-16 h-16 rounded-full bg-red-100 dark:bg-red-900/30 flex items-center justify-center">
          <ShieldX className="w-8 h-8 text-red-600 dark:text-red-400" />
        </div>
        <h2 className="text-2xl font-bold text-foreground">Access Denied</h2>
        <p className="text-muted-foreground max-w-sm">
          You do not have administrator privileges to view this page.
        </p>
        <Button
          onClick={() => navigate("/")}
          className="bg-blue-600 hover:bg-blue-700 text-white"
        >
          Go to Dashboard
        </Button>
      </div>
    );
  }

  // ── Handlers ──────────────────────────────────────────────────────────────

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    setFormData((prev) => ({ ...prev, [name]: value }));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (
      !formData.symbol ||
      !formData.name ||
      !formData.price ||
      !formData.quantity
    ) {
      toast.error("Please fill in all fields");
      return;
    }

    try {
      const payload = {
        symbol: formData.symbol.toUpperCase(),
        name: formData.name,
        price: parseFloat(formData.price),
        quantity: parseInt(formData.quantity, 10),
      };

      if (editingId) {
        await adminApi.updateStock(editingId, {
          price: payload.price,
          quantity: payload.quantity,
        });
        toast.success("Stock updated successfully");
      } else {
        await adminApi.createStock(payload);
        toast.success("Stock created successfully");
      }

      setFormData({
        symbol: "",
        name: "",
        price: "",
        quantity: "",
      });
      setEditingId(null);
      fetchStocks();
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : "Action failed");
    }
  };

  const handleEdit = (stock: Stock) => {
    setEditingId(stock.stock_id);
    setFormData({
      symbol: stock.symbol,
      name: stock.name,
      price: stock.price.toString(),
      quantity: stock.quantity.toString(),
    });
  };

  const handleDelete = async (stockId: string) => {
    if (!window.confirm("Are you sure you want to delete this stock?")) return;
    try {
      await adminApi.deleteStock(stockId);
      toast.success("Stock deleted");
      fetchStocks();
    } catch (err: unknown) {
      toast.error(
        err instanceof Error ? err.message : "Failed to delete stock",
      );
    }
  };

  const handleStartMarket = async () => {
    try {
      setMarketLoading(true);
      const res = await adminApi.startMarket();
      if (res?.status) setMarketStatus(res.status);
      toast.success("Market opened for trading!");
      await fetchMarketStatus();
    } catch (err: unknown) {
      toast.error(
        err instanceof Error ? err.message : "Failed to start market",
      );
    } finally {
      setMarketLoading(false);
    }
  };

  const handleStopMarket = async () => {
    try {
      setMarketLoading(true);
      const res = await adminApi.stopMarket();
      if (res?.status) setMarketStatus(res.status);
      toast.success("Market closed. No trading allowed.");
      await fetchMarketStatus();
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : "Failed to stop market");
    } finally {
      setMarketLoading(false);
    }
  };

  // ── Render ─────────────────────────────────────────────────────────────────

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold text-foreground">Admin Portal</h1>
        <p className="text-muted-foreground mt-1">
          Manage stocks and market operations
        </p>
      </div>

      {/* ── Market Control ── */}
      <Card className="border-2 border-blue-500/50 bg-linear-to-r from-blue-50 to-transparent dark:from-blue-950/30 dark:to-transparent">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <PowerCircle className="w-5 h-5 text-blue-600" />
            Market Control
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 p-4 bg-muted/50 rounded-lg border border-border">
            <div className="flex-1">
              <div className="text-lg font-semibold text-foreground">
                Market Status:{" "}
                <span
                  className={
                    marketStatus?.is_open
                      ? "text-green-600 dark:text-green-400"
                      : "text-red-600 dark:text-red-400"
                  }
                >
                  {marketStatus?.is_open ? "OPEN" : "CLOSED"}
                </span>
              </div>
              <p className="text-sm text-muted-foreground mt-1">
                {marketStatus?.is_open
                  ? `Trading started at ${
                      marketStatus.opened_at
                        ? new Date(marketStatus.opened_at).toLocaleString()
                        : "N/A"
                    }`
                  : marketStatus?.closed_at
                    ? `Trading stopped at ${new Date(
                        marketStatus.closed_at,
                      ).toLocaleString()}`
                    : "Market is currently closed"}
              </p>
            </div>

            <div className="flex gap-3 w-full sm:w-auto">
              <Button
                onClick={handleStartMarket}
                disabled={marketStatus?.is_open || marketLoading}
                className="flex-1 sm:flex-none bg-green-600 hover:bg-green-700 text-white"
              >
                {marketLoading && (
                  <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                )}
                <Play className="w-4 h-4 mr-2" />
                Start Market
              </Button>

              <Button
                onClick={handleStopMarket}
                disabled={!marketStatus?.is_open || marketLoading}
                className="flex-1 sm:flex-none bg-red-600 hover:bg-red-700 text-white"
              >
                {marketLoading && (
                  <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                )}
                <Square className="w-4 h-4 mr-2" />
                Stop Market
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* ── Stock Management ── */}
      <div>
        <h2 className="text-xl font-semibold text-foreground mb-4">
          Stock Management
        </h2>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Form */}
        <Card className="lg:col-span-1">
          <CardHeader>
            <CardTitle>
              {editingId ? "Edit Stock" : "Create New Stock"}
            </CardTitle>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="symbol">Symbol</Label>
                <Input
                  id="symbol"
                  name="symbol"
                  value={formData.symbol}
                  onChange={handleInputChange}
                  disabled={!!editingId}
                  placeholder="e.g. AAPL"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="name">Company Name</Label>
                <Input
                  id="name"
                  name="name"
                  value={formData.name}
                  onChange={handleInputChange}
                  disabled={!!editingId}
                  placeholder="e.g. Apple Inc."
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="price">Price</Label>
                <Input
                  id="price"
                  name="price"
                  type="number"
                  step="0.01"
                  value={formData.price}
                  onChange={handleInputChange}
                  placeholder="150.00"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="quantity">Quantity (Shares)</Label>
                <Input
                  id="quantity"
                  name="quantity"
                  type="number"
                  value={formData.quantity}
                  onChange={handleInputChange}
                  placeholder="1000"
                />
              </div>
              <div className="flex gap-2">
                <Button
                  type="submit"
                  className="w-full bg-blue-600 hover:bg-blue-700"
                >
                  {editingId ? "Update Stock" : "Create Stock"}
                </Button>
                {editingId && (
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => {
                      setEditingId(null);
                      setFormData({
                        symbol: "",
                        name: "",
                        price: "",
                        quantity: "",
                      });
                    }}
                  >
                    Cancel
                  </Button>
                )}
              </div>
            </form>
          </CardContent>
        </Card>

        {/* Stock List */}
        <Card className="lg:col-span-2">
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle>Existing Stocks</CardTitle>
            <Button variant="outline" size="sm" onClick={fetchStocks}>
              Refresh
            </Button>
          </CardHeader>
          <CardContent>
            {loading ? (
              <div className="flex items-center justify-center p-8">
                <Loader2 className="w-8 h-8 animate-spin text-blue-500" />
              </div>
            ) : stocks.length === 0 ? (
              <div className="text-center py-8 text-muted-foreground">
                No stocks found in the database.
              </div>
            ) : (
              <div className="space-y-3">
                {stocks.map((stock) => (
                  <div
                    key={stock.stock_id}
                    className="flex items-center justify-between p-4 bg-muted/50 rounded-lg border border-border"
                  >
                    <div className="flex-1">
                      <div className="flex items-center gap-2">
                        <span className="font-bold text-foreground">
                          {stock.symbol}
                        </span>
                        <span className="text-sm text-muted-foreground">
                          {stock.name}
                        </span>
                      </div>
                      <div className="text-sm text-muted-foreground mt-1">
                        Price: {formatPrice(stock.price)} | Qty: {stock.quantity}
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => handleEdit(stock)}
                        className="text-blue-500 hover:text-blue-600 hover:bg-blue-50 dark:hover:bg-blue-900/50"
                      >
                        <Edit2 className="w-4 h-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => handleDelete(stock.stock_id)}
                        className="text-red-500 hover:text-red-600 hover:bg-red-50 dark:hover:bg-red-900/50"
                      >
                        <Trash2 className="w-4 h-4" />
                      </Button>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
