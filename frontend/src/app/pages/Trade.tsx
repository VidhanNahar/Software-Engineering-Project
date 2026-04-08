import { useState, useEffect, useRef } from "react";
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
import { formatPrice } from "../utils/currency";
import { toast } from "sonner";
import { isKycVerified } from "../utils/auth";
import { stockApi, transactionApi, walletApi, portfolioApi, adminApi, ordersApi, WS_STOCKS_URL } from "../api";

export default function Trade() {
  const location = useLocation();
  const navigate = useNavigate();
  const kycStatus = isKycVerified();
  const state = location.state as {
    symbol?: string;
    type?: "buy" | "sell";
  } | null;

  const [stocks, setStocks] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [walletBalance, setWalletBalance] = useState(0);
  const [holdings, setHoldings] = useState<any[]>([]);
  const [pendingOrders, setPendingOrders] = useState<any[]>([]);

  const [orderType, setOrderType] = useState<"buy" | "sell">(
    state?.type || "buy",
  );
  const [selectedStockId, setSelectedStockId] = useState("");
  const [quantity, setQuantity] = useState("10");
  const [orderMode, setOrderMode] = useState<"market" | "limit">("market");
  const [limitPrice, setLimitPrice] = useState("");

  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isSuccessDialogOpen, setIsSuccessDialogOpen] = useState(false);
  const [successMessage, setSuccessMessage] = useState("");
  const [cancelingOrderId, setCancelingOrderId] = useState<string | null>(null);
  const [isMarketOpen, setIsMarketOpen] = useState(true);
  const isMarketOpenRef = useRef(true);

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
        setPendingOrders(portfolioRes?.pending_orders || []);

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
        console.log("💫 Trade WebSocket connected");
      };

      ws.onmessage = (evt) => {
        try {
          const data = JSON.parse(evt.data);

          // Handle market status updates
          if (data.type === "market_status") {
            setIsMarketOpen(data.market_open);
            isMarketOpenRef.current = data.market_open;  // Update ref immediately
            if (data.market_open === false) {
              console.log("🔒 MARKET CLOSED - Trading disabled");
            } else {
              console.log("🔓 MARKET OPENED - Trading enabled");
            }
            return;
          }

          // Check if market is open using ref (synchronous check)
          if (!isMarketOpenRef.current) {
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

  const refreshPortfolioData = async () => {
    const [walletRes, portfolioRes] = await Promise.all([
      walletApi.get().catch(() => ({ balance: 0 })),
      portfolioApi.get().catch(() => ({ holdings: [], pending_orders: [] })),
    ]);
    setWalletBalance(walletRes?.balance || 0);
    setHoldings(portfolioRes?.holdings || []);
    setPendingOrders(portfolioRes?.pending_orders || []);
  };

  const handleCancelPendingOrder = async (orderId: string) => {
    try {
      setCancelingOrderId(orderId);
      await ordersApi.cancelPendingOrder(orderId);
      toast.success("Pending order cancelled");
      await refreshPortfolioData();
    } catch (err: any) {
      toast.error(err?.message || "Failed to cancel pending order");
    } finally {
      setCancelingOrderId(null);
    }
  };

  const handlePlaceOrder = async () => {
    if (!selectedStockId) {
      toast.error("Please select a stock");
      return;
    }

    if (!quantity || qty <= 0) {
      toast.error("Please enter a valid quantity");
      return;
    }

    const limitPriceNum = parseFloat(limitPrice);
    if (orderMode === "limit" && (!limitPrice || limitPriceNum <= 0)) {
      toast.error("Please enter a valid limit price (must be positive)");
      return;
    }

    // Basic Validations
    if (orderType === "buy" && totalWithFee > walletBalance) {
      toast.error("Insufficient funds for this trade");
      return;
    }

    if (orderType === "sell") {
      const ownedQuantity =
        holdings.find((h) => h.stock_id === selectedStockId)?.available_qty ??
        holdings.find((h) => h.stock_id === selectedStockId)?.quantity ??
        0;
      if (qty > ownedQuantity) {
        toast.error("You do not own enough shares to sell");
        return;
      }
    }

    if (!kycStatus) {
      toast.error("KYC verification required for trading");
      navigate("/profile");
      return;
    }

    setIsSubmitting(true);
    try {
      const payload = {
        stock_id: selectedStockId,
        quantity: qty,
        price_per_stock: orderMode === "limit" ? limitPriceNum : price,
        order_type: orderMode === "limit" ? "LIMIT" : "MARKET",
        time_in_force: "DAY",
      };

      let responseData: any;
      if (orderType === "buy") {
        responseData = await transactionApi.buy(payload);
      } else {
        responseData = await transactionApi.sell(payload);
      }

      // Check if order is pending or executed
      if (responseData?.status === "PENDING" || responseData?.message?.includes("pending")) {
        // Limit order was created but not executed
        setSuccessMessage(
          `📋 Limit order created and pending\n\n` +
          `${orderType === "buy" ? "Buy" : "Sell"} ${qty} shares at ₹${limitPriceNum}\n` +
          `Order will execute when price ${orderType === "buy" ? "drops to" : "rises to"} ₹${limitPriceNum} or ${orderType === "buy" ? "lower" : "higher"}`
        );
        toast.success("Pending limit order created! Check your pending orders.");
      } else {
        // Order was executed immediately
        const orderTypeText = orderMode === "limit" ? `limit order at ₹${limitPriceNum}` : "market order";
        setSuccessMessage(`Successfully placed ${orderType} ${orderTypeText} for ${qty} shares!`);
        toast.success(`${orderType === "buy" ? "Buy" : "Sell"} order executed!`);
      }

      setIsSuccessDialogOpen(true);

      // Refresh wallet & portfolio after trade
      await refreshPortfolioData();

      setTimeout(() => {
        navigate("/portfolio");
      }, 3000);
      setQuantity("");
    } catch (err: any) {
      // Parse error message from response
      let errorMsg = err.message || "Failed to execute trade";
      try {
        if (err.message.includes("within")) {
          // Price validation error from backend
          errorMsg = err.message;
        }
      } catch {}
      toast.error(errorMsg);
    } finally {
      setIsSubmitting(false);
    }
  };

  if (loading) {
    return (
      <div className="flex justify-center items-center h-full">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
      </div>
    );
  }

  const selectedHolding = holdings.find((h) => h.stock_id === selectedStockId);
  const ownedQuantity = selectedHolding?.quantity || 0;
  const availableQuantity = selectedHolding?.available_qty ?? ownedQuantity;
  const lockedQuantity = selectedHolding?.pending_sell_qty || 0;
  const pendingLimitOrders = pendingOrders.filter(
    (o) => o.status === "PENDING" || o.status === "PARTIALLY_FILLED",
  );

  const getStockDisplay = (stockId: string) => {
    const stock = stocks.find((s) => s.stock_id === stockId);
    return stock ? `${stock.symbol} - ${stock.name}` : stockId;
  };

  return (
    <div className="max-w-6xl mx-auto space-y-6">
      <Dialog open={isSuccessDialogOpen} onOpenChange={() => {}}>
        <DialogContent className="sm:max-w-md bg-card border-border text-foreground flex flex-col items-center justify-center py-10 [&>button]:hidden">
          <div className="bg-green-500/10 p-3 rounded-full mb-4">
            <CheckCircle2 className="w-12 h-12 text-green-500" />
          </div>
          <DialogTitle className="text-2xl font-bold text-center mb-2">
            Trade Successful
          </DialogTitle>
          <DialogDescription className="text-muted-foreground text-center text-lg">
            {successMessage}
          </DialogDescription>
        </DialogContent>
      </Dialog>

      {!isMarketOpen && (
        <div className="bg-red-900/20 border border-red-500/50 rounded-lg p-4 text-center">
          <p className="text-red-400 font-semibold text-lg">🔒 Market Closed</p>
          <p className="text-red-300 text-sm mt-1">Trading is disabled. Market will open for new orders when admin resumes trading.</p>
        </div>
      )}

      <div>
        <h1 className="text-3xl font-bold">Trade</h1>
        <p className="text-muted-foreground mt-1">Place buy or sell orders</p>
      </div>

      {!kycStatus && (
        <Card className="bg-amber-50 dark:bg-amber-900/10 border-amber-200 dark:border-amber-800">
          <CardContent className="p-4 flex items-center gap-4">
            <div className="bg-amber-100 dark:bg-amber-900/30 p-2 rounded-full">
              <AlertCircle className="w-5 h-5 text-amber-600 dark:text-amber-400" />
            </div>
            <div className="flex-1">
              <p className="font-semibold text-amber-900 dark:text-amber-200">
                KYC Verification Required
              </p>
              <p className="text-sm text-amber-700 dark:text-amber-400">
                You must complete your identity verification to start trading.
              </p>
            </div>
            <Button
              variant="outline"
              size="sm"
              className="border-amber-300 dark:border-amber-700 hover:bg-amber-100 dark:hover:bg-amber-900/50"
              onClick={() => navigate("/profile")}
            >
              Complete KYC
            </Button>
          </CardContent>
        </Card>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="lg:col-span-2 space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Order Details</CardTitle>
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
                      <Label htmlFor="symbol">
                        Symbol
                      </Label>
                      <Select
                        value={selectedStockId}
                        onValueChange={setSelectedStockId}
                      >
                        <SelectTrigger className="bg-input-background border-border">
                          <SelectValue placeholder="Select stock" />
                        </SelectTrigger>
                        <SelectContent className="bg-popover border-border">
                          {stocks.map((s) => (
                            <SelectItem key={s.stock_id} value={s.stock_id}>
                              {s.symbol} - {s.name}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>

                    <div className="space-y-2">
                      <Label htmlFor="orderMode">
                        Order Type
                      </Label>
                      <Select
                        value={orderMode}
                        onValueChange={(v: "market" | "limit") =>
                          setOrderMode(v)
                        }
                      >
                        <SelectTrigger className="bg-input-background border-border">
                          <SelectValue placeholder="Order Type" />
                        </SelectTrigger>
                        <SelectContent className="bg-popover border-border text-foreground">
                          <SelectItem value="market">Market Order</SelectItem>
                          <SelectItem value="limit">Limit Order</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>

                    <div className="space-y-2">
                      <Label htmlFor="quantity">
                        Quantity
                      </Label>
                      <Input
                        id="quantity"
                        type="number"
                        min="1"
                        value={quantity}
                        onChange={(e) => setQuantity(e.target.value)}
                        className="bg-input-background border-border"
                      />
                      {orderType === "sell" && (
                        <p className="text-xs text-muted-foreground">
                          Available to sell: {availableQuantity}
                          {lockedQuantity > 0 ? ` (Locked in pending limit sell: ${lockedQuantity})` : ""}
                        </p>
                      )}
                    </div>

                    {orderMode === "limit" && (
                      <div className="space-y-2">
                        <Label htmlFor="limitPrice">
                          Limit Price
                        </Label>
                        <div className="relative">
                          <span className="absolute left-3 top-1/2 transform -translate-y-1/2 text-muted-foreground">
                            {formatPrice(0).charAt(0)}
                          </span>
                          <Input
                            id="limitPrice"
                            type="number"
                            min="0.01"
                            step="0.01"
                            value={limitPrice}
                            onChange={(e) => setLimitPrice(e.target.value)}
                            className="pl-8 bg-input-background border-border"
                            placeholder="0.00"
                          />
                        </div>
                      </div>
                    )}
                  </div>
                </div>
              </Tabs>
            </CardContent>
          </Card>
        </div>

        <div className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Order Summary</CardTitle>
            </CardHeader>
            <CardContent>
              {selectedStock ? (
                <div className="space-y-6">
                  <div>
                    <p className="text-sm text-muted-foreground">Stock</p>
                    <p className="text-xl font-bold">
                      {selectedStock.symbol}
                    </p>
                    <p className="text-sm text-muted-foreground">
                      {selectedStock.name}
                    </p>
                  </div>

                    <div className="space-y-3 pt-4 border-t border-border">
                    <div className="flex justify-between text-sm">
                      <span className="text-muted-foreground">Current Price</span>
                      <span className="font-semibold">
                        {formatPrice(currentPrice)}
                      </span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span className="text-muted-foreground">Quantity</span>
                      <span className="font-semibold">{qty}</span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span className="text-muted-foreground">Order Price</span>
                      <span className="font-semibold">
                        {formatPrice(price)}
                      </span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span className="text-muted-foreground">Subtotal</span>
                      <span className="font-semibold">
                        {formatPrice(total)}
                      </span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span className="text-muted-foreground">Est. Fee (0.1%)</span>
                      <span className="text-muted-foreground">
                        {formatPrice(estimatedFee)}
                      </span>
                    </div>
                  </div>

                  <div className="pt-4 border-t border-border">
                    <div className="flex justify-between items-center mb-4">
                      <span className="text-lg font-semibold">
                        Total
                      </span>
                      <span className="text-2xl font-bold">
                        {formatPrice(totalWithFee)}
                      </span>
                    </div>

                    <div className="flex justify-between text-sm mb-6 text-muted-foreground">
                      <span>Available Buying Power</span>
                      <span
                        className={
                          orderType === "buy" && totalWithFee > walletBalance
                            ? "text-red-500 font-bold"
                            : ""
                        }
                      >
                        {formatPrice(walletBalance)}
                      </span>
                    </div>

                    <Button
                      className={`w-full text-white font-bold text-lg h-12 ${
                        orderType === "buy"
                          ? "bg-green-600 hover:bg-green-700"
                          : "bg-red-600 hover:bg-red-700"
                      }`}
                      onClick={handlePlaceOrder}
                      disabled={isSubmitting || !isMarketOpen}
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
                <p className="text-muted-foreground text-center py-8">
                  Select a stock to view summary
                </p>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardContent className="p-4 bg-blue-50/5 dark:bg-blue-900/10 text-blue-800 dark:text-blue-200 text-sm flex items-start gap-3 rounded-lg border border-blue-100 dark:border-blue-800">
              <AlertCircle className="w-5 h-5 flex-shrink-0 mt-0.5" />
              <div>
                <p className="font-semibold mb-1">
                  Trading Information
                </p>
                <p className="text-muted-foreground">
                  Market orders execute at current market price. Limit orders
                  execute only at your specified price or better.
                </p>
              </div>
            </CardContent>
          </Card>

          {pendingLimitOrders.length > 0 && (
            <Card>
              <CardHeader>
                <CardTitle>Pending Limit Orders</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-2 text-sm">
                  {pendingLimitOrders.map((order) => (
                    <div key={order.order_id} className="flex items-center justify-between rounded-md border border-border p-2">
                      <div>
                        <p className="font-medium">{order.order_type} • Qty: {order.quantity - order.filled_quantity}</p>
                        <p className="text-muted-foreground">{getStockDisplay(order.stock_id)}</p>
                        <p className="text-muted-foreground">Limit: {formatPrice(order.limit_price)}</p>
                      </div>
                      <div className="flex items-center gap-3">
                        <p className="text-muted-foreground">{order.status}</p>
                        <Button
                          type="button"
                          variant="outline"
                          size="sm"
                          disabled={cancelingOrderId === order.order_id || !isMarketOpen}
                          onClick={() => handleCancelPendingOrder(order.order_id)}
                        >
                          {!isMarketOpen
                            ? "Market Closed"
                            : cancelingOrderId === order.order_id
                              ? "Cancelling..."
                              : "Cancel"}
                        </Button>
                      </div>
                    </div>
                  ))}
                </div>
              </CardContent>
            </Card>
          )}
        </div>
      </div>
    </div>
  );
}
