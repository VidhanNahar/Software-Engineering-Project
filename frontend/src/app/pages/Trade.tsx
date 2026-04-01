import { useState, useEffect } from "react";
import { useLocation, useNavigate } from "react-router";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "../components/ui/tabs";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "../components/ui/dialog";
import { TrendingUp, TrendingDown, AlertCircle, Loader2, CheckCircle2 } from "lucide-react";
import { toast } from "sonner";
import { stockApi, transactionApi, walletApi, portfolioApi } from "../api";

export default function Trade() {
  const location = useLocation();
  const navigate = useNavigate();
  const state = location.state as {
    symbol?: string;
    type?: "buy" | "sell";
  } | null;

  const [stocks, setStocks] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [walletBalance, setWalletBalance] = useState(0);
  const [holdings, setHoldings] = useState<any[]>([]);

  const [orderType, setOrderType] = useState<"buy" | "sell">(
    state?.type || "buy",
  );
  const [selectedStockId, setSelectedStockId] = useState("");
  const [quantity, setQuantity] = useState("10");
  const [orderMode, setOrderMode] = useState<"market" | "limit">("market");
  const [limitPrice, setLimitPrice] = useState("");
  const [timeInForce, setTimeInForce] = useState("day");

  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isSuccessDialogOpen, setIsSuccessDialogOpen] = useState(false);
  const [successMessage, setSuccessMessage] = useState("");

  useEffect(() => {
    const fetchData = async () => {
      try {
        const [stocksRes, walletRes, portfolioRes] = await Promise.all([
          stockApi.getAll(),
          walletApi.get().catch(() => ({ balance: 0 })),
          portfolioApi.get().catch(() => ({ holdings: [] })),
        ]);

        const loadedStocks = stocksRes.stocks || [];
        setStocks(loadedStocks);
        setWalletBalance(walletRes?.balance || 0);
        setHoldings(portfolioRes?.holdings || []);

        if (loadedStocks.length > 0) {
          const preselected = state?.symbol
            ? loadedStocks.find((s: any) => s.symbol === state.symbol)
            : loadedStocks[0];
          setSelectedStockId(preselected?.stock_id || loadedStocks[0].stock_id);
        }
      } catch (e) {
        toast.error("Failed to load trading data");
      } finally {
        setLoading(false);
      }
    };
    fetchData();

    // Establish WebSocket connection for real-time price updates
    let ws: WebSocket | null = null;
    try {
      const wsProtocol = window.location.protocol === "https:" ? "wss" : "ws";
      ws = new WebSocket(`${wsProtocol}://${window.location.host}/ws/stocks`);

      ws.onopen = () => {
        console.log("💫 Trade WebSocket connected");
      };

      ws.onmessage = (evt) => {
        try {
          const data = JSON.parse(evt.data);

          // If market is closed, don't update prices
          if (data.market_open === false) {
            console.debug("📊 Market closed, freezing Trade prices");
            return;
          }

          // Update stocks with real-time prices
          if (data.type === "stock_tick" || data.type === "stocks_snapshot") {
            const incomingStocks = data.stocks || data.ticks || [];

            setStocks((prev) =>
              prev.map((stock) => {
                const updated = incomingStocks.find(
                  (s: Record<string, unknown>) => s.symbol === stock.symbol
                );
                if (updated && typeof updated.price === "number") {
                  return {
                    ...stock,
                    price: updated.price,
                  };
                }
                return stock;
              })
            );
          }
        } catch (e) {
          // Ignore malformed WebSocket payloads
        }
      };

      ws.onerror = () => {
        console.log("⚠️ Trade WebSocket error");
      };

      ws.onclose = () => {
        console.log("🔌 Trade WebSocket disconnected");
      };
    } catch (error) {
      console.error("Failed to connect to WebSocket", error);
    }

    return () => {
      if (ws) {
        ws.close();
      }
    };
  }, [state?.symbol]);

  const selectedStock = stocks.find((s) => s.stock_id === selectedStockId);
  const qty = parseFloat(quantity) || 0;
  const currentPrice = selectedStock?.price || 0;
  const price =
    orderMode === "limit" && limitPrice ? parseFloat(limitPrice) : currentPrice;

  const total = qty * price;
  const estimatedFee = total * 0.001; // 0.1% fee
  const totalWithFee = total + estimatedFee;

  const handlePlaceOrder = async () => {
    if (!selectedStockId) {
      toast.error("Please select a stock");
      return;
    }

    if (!quantity || qty <= 0) {
      toast.error("Please enter a valid quantity");
      return;
    }

    if (orderMode === "limit" && (!limitPrice || price <= 0)) {
      toast.error("Please enter a valid limit price");
      return;
    }

    // Basic Validations
    if (orderType === "buy" && totalWithFee > walletBalance) {
      toast.error("Insufficient funds for this trade");
      return;
    }

    if (orderType === "sell") {
      const ownedQuantity =
        holdings.find((h) => h.stock_id === selectedStockId)?.quantity || 0;
      if (qty > ownedQuantity) {
        toast.error("You do not own enough shares to sell");
        return;
      }
    }

    setIsSubmitting(true);
    try {
      const payload = {
        stock_id: selectedStockId,
        quantity: qty,
        price_per_stock: price, // Handled automatically in market orders if omitted depending on backend design
      };

      if (orderType === "buy") {
        await transactionApi.buy(payload);
        setSuccessMessage(`Successfully placed buy order for ${qty} shares!`);
      } else {
        await transactionApi.sell(payload);
        setSuccessMessage(`Successfully placed sell order for ${qty} shares!`);
      }

      setIsSuccessDialogOpen(true);

      setTimeout(() => {
        navigate("/portfolio");
      }, 3000);

      // Refresh wallet & portfolio after trade
      const [walletRes, portfolioRes] = await Promise.all([
        walletApi.get().catch(() => ({ balance: walletBalance })),
        portfolioApi.get().catch(() => ({ holdings })),
      ]);
      setWalletBalance(walletRes?.balance || 0);
      setHoldings(portfolioRes?.holdings || []);
      setQuantity("");
    } catch (err: any) {
      toast.error(err.message || "Failed to execute trade");
    } finally {
      setIsSubmitting(false);
    }
  };

  if (loading) {
    return (
      <div className="flex justify-center items-center h-full text-white">
        <Loader2 className="w-8 h-8 animate-spin" />
      </div>
    );
  }

  const ownedQuantity =
    holdings.find((h) => h.stock_id === selectedStockId)?.quantity || 0;

  return (
    <div className="max-w-6xl mx-auto space-y-6">
      <Dialog open={isSuccessDialogOpen} onOpenChange={() => {}}>
        <DialogContent className="sm:max-w-md bg-gray-900 border-gray-700 text-white flex flex-col items-center justify-center py-10 [&>button]:hidden">
          <div className="bg-green-500/10 p-3 rounded-full mb-4">
            <CheckCircle2 className="w-12 h-12 text-green-500" />
          </div>
          <DialogTitle className="text-2xl font-bold text-center mb-2">
            Trade Successful
          </DialogTitle>
          <DialogDescription className="text-gray-300 text-center text-lg">
            {successMessage}
          </DialogDescription>
        </DialogContent>
      </Dialog>

      <div>
        <h1 className="text-3xl font-bold text-white">Trade</h1>
        <p className="text-gray-300 mt-1">Place buy or sell orders</p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="lg:col-span-2 space-y-6">
          <Card className="text-white">
            <CardHeader>
              <CardTitle className="text-white">Order Details</CardTitle>
            </CardHeader>
            <CardContent>
              <Tabs
                value={orderType}
                onValueChange={(v) => setOrderType(v as "buy" | "sell")}
                className="w-full"
              >
                <TabsList className="grid w-full grid-cols-2 mb-6">
                  <TabsTrigger
                    value="buy"
                    className="data-[state=active]:bg-green-600 data-[state=active]:text-white"
                  >
                    Buy
                  </TabsTrigger>
                  <TabsTrigger
                    value="sell"
                    className="data-[state=active]:bg-red-600 data-[state=active]:text-white"
                  >
                    Sell
                  </TabsTrigger>
                </TabsList>

                <div className="space-y-6">
                  <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-2">
                      <Label className="text-white" htmlFor="symbol">
                        Symbol
                      </Label>
                      <Select
                        value={selectedStockId}
                        onValueChange={setSelectedStockId}
                      >
                        <SelectTrigger className="bg-gray-800 text-white border-gray-700">
                          <SelectValue placeholder="Select stock" />
                        </SelectTrigger>
                        <SelectContent className="bg-gray-800 border-gray-700 text-white">
                          {stocks.map((s) => (
                            <SelectItem key={s.stock_id} value={s.stock_id}>
                              {s.symbol} - {s.name}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>

                    <div className="space-y-2">
                      <Label className="text-white" htmlFor="orderMode">
                        Order Type
                      </Label>
                      <Select
                        value={orderMode}
                        onValueChange={(v: "market" | "limit") =>
                          setOrderMode(v)
                        }
                      >
                        <SelectTrigger className="bg-gray-800 text-white border-gray-700">
                          <SelectValue placeholder="Order Type" />
                        </SelectTrigger>
                        <SelectContent className="bg-gray-800 border-gray-700 text-white">
                          <SelectItem value="market">Market Order</SelectItem>
                          <SelectItem value="limit">Limit Order</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>

                    <div className="space-y-2">
                      <Label className="text-white" htmlFor="quantity">
                        Quantity
                      </Label>
                      <Input
                        id="quantity"
                        type="number"
                        min="1"
                        value={quantity}
                        onChange={(e) => setQuantity(e.target.value)}
                        className="bg-gray-800 text-white border-gray-700"
                      />
                      {orderType === "sell" && (
                        <p className="text-xs text-gray-400">
                          Available to sell: {ownedQuantity}
                        </p>
                      )}
                    </div>

                    {orderMode === "limit" && (
                      <div className="space-y-2">
                        <Label className="text-white" htmlFor="limitPrice">
                          Limit Price
                        </Label>
                        <div className="relative">
                          <span className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400">
                            $
                          </span>
                          <Input
                            id="limitPrice"
                            type="number"
                            min="0.01"
                            step="0.01"
                            value={limitPrice}
                            onChange={(e) => setLimitPrice(e.target.value)}
                            className="pl-8 bg-gray-800 text-white border-gray-700"
                            placeholder="0.00"
                          />
                        </div>
                      </div>
                    )}

                    <div className="space-y-2">
                      <Label className="text-white" htmlFor="timeInForce">
                        Time in Force
                      </Label>
                      <Select
                        value={timeInForce}
                        onValueChange={setTimeInForce}
                      >
                        <SelectTrigger className="bg-gray-800 text-white border-gray-700">
                          <SelectValue placeholder="Time in Force" />
                        </SelectTrigger>
                        <SelectContent className="bg-gray-800 border-gray-700 text-white">
                          <SelectItem value="day">Day</SelectItem>
                          <SelectItem value="gtc">
                            Good 'Til Cancelled
                          </SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                  </div>
                </div>
              </Tabs>
            </CardContent>
          </Card>
        </div>

        <div className="space-y-6">
          <Card className="text-white">
            <CardHeader>
              <CardTitle className="text-white">Order Summary</CardTitle>
            </CardHeader>
            <CardContent>
              {selectedStock ? (
                <div className="space-y-6">
                  <div>
                    <p className="text-sm text-gray-300">Stock</p>
                    <p className="text-xl font-bold text-white">
                      {selectedStock.symbol}
                    </p>
                    <p className="text-sm text-gray-300">
                      {selectedStock.name}
                    </p>
                  </div>

                  <div className="space-y-3 pt-4 border-t border-gray-800">
                    <div className="flex justify-between text-sm">
                      <span className="text-gray-300">Current Price</span>
                      <span className="font-semibold text-white">
                        ${currentPrice.toFixed(2)}
                      </span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span className="text-gray-300">Quantity</span>
                      <span className="font-semibold text-white">{qty}</span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span className="text-gray-300">Order Price</span>
                      <span className="font-semibold text-white">
                        ${price.toFixed(2)}
                      </span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span className="text-gray-300">Subtotal</span>
                      <span className="font-semibold text-white">
                        ${total.toFixed(2)}
                      </span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span className="text-gray-300">Est. Fee (0.1%)</span>
                      <span className="text-gray-300">
                        ${estimatedFee.toFixed(2)}
                      </span>
                    </div>
                  </div>

                  <div className="pt-4 border-t border-gray-800">
                    <div className="flex justify-between items-center mb-4">
                      <span className="text-lg font-semibold text-white">
                        Total
                      </span>
                      <span className="text-2xl font-bold text-white">
                        ${totalWithFee.toFixed(2)}
                      </span>
                    </div>

                    <div className="flex justify-between text-sm mb-6 text-gray-400">
                      <span>Available Buying Power</span>
                      <span
                        className={
                          orderType === "buy" && totalWithFee > walletBalance
                            ? "text-red-500 font-bold"
                            : ""
                        }
                      >
                        ${walletBalance.toFixed(2)}
                      </span>
                    </div>

                    <Button
                      className={`w-full text-white font-bold text-lg h-12 ${
                        orderType === "buy"
                          ? "bg-green-600 hover:bg-green-700"
                          : "bg-red-600 hover:bg-red-700"
                      }`}
                      onClick={handlePlaceOrder}
                      disabled={isSubmitting}
                    >
                      {isSubmitting ? (
                        <Loader2 className="h-6 w-6 animate-spin" />
                      ) : orderType === "buy" ? (
                        "Place Buy Order"
                      ) : (
                        "Place Sell Order"
                      )}
                    </Button>
                  </div>
                </div>
              ) : (
                <p className="text-gray-400 text-center py-8">
                  Select a stock to view summary
                </p>
              )}
            </CardContent>
          </Card>

          <Card className="text-white">
            <CardContent className="p-4 bg-blue-50/5 dark:bg-blue-900/10 text-blue-800 dark:text-blue-200 text-sm flex items-start gap-3 rounded-lg border border-blue-100 dark:border-blue-800">
              <AlertCircle className="w-5 h-5 flex-shrink-0 mt-0.5" />
              <div>
                <p className="font-semibold text-white mb-1">
                  Trading Information
                </p>
                <p className="text-gray-300">
                  Market orders execute at current market price. Limit orders
                  execute only at your specified price or better.
                </p>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
