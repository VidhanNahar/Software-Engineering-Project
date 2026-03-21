import { useState, useEffect } from "react";
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
import { adminApi, stockApi } from "../api";
import { Trash2, Edit2, Loader2, Plus } from "lucide-react";

export default function Admin() {
  const [stocks, setStocks] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
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
    } catch (err) {
      toast.error("Failed to load stocks");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchStocks();
  }, []);

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

      setFormData({ symbol: "", name: "", price: "", quantity: "" });
      setEditingId(null);
      fetchStocks();
    } catch (err: any) {
      toast.error(err.message || "Action failed");
    }
  };

  const handleEdit = (stock: any) => {
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
    } catch (err: any) {
      toast.error(err.message || "Failed to delete stock");
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold text-foreground">Admin Portal</h1>
        <p className="text-muted-foreground mt-1">
          Manage stocks in the system
        </p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Form Card */}
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
                <Label htmlFor="price">Price ($)</Label>
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

        {/* Stock List Card */}
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
                        Price: ${stock.price.toFixed(2)} | Qty: {stock.quantity}
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
