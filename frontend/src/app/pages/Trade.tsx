import { useState, useEffect } from "react";
import { useLocation } from "react-router";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../components/ui/tabs";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../components/ui/select";
import { TrendingUp, TrendingDown, AlertCircle } from "lucide-react";
import { mockStocks } from "../utils/mockData";
import { toast } from "sonner";

export default function Trade() {
  const location = useLocation();
  const state = location.state as { symbol?: string; type?: "buy" | "sell" } | null;

  const [orderType, setOrderType] = useState<"buy" | "sell">(state?.type || "buy");
  const [symbol, setSymbol] = useState(state?.symbol || "AAPL");
  const [quantity, setQuantity] = useState("10");
  const [orderMode, setOrderMode] = useState<"market" | "limit">("market");
  const [limitPrice, setLimitPrice] = useState("");
  const [timeInForce, setTimeInForce] = useState("day");

  const stock = mockStocks[symbol];
  const qty = parseFloat(quantity) || 0;
  const price = orderMode === "limit" && limitPrice ? parseFloat(limitPrice) : stock?.price || 0;
  const total = qty * price;
  const estimatedFee = total * 0.001; // 0.1% fee
  const totalWithFee = total + estimatedFee;

  const handlePlaceOrder = () => {
    if (!quantity || qty <= 0) {
      toast.error("Please enter a valid quantity");
      return;
    }

    if (orderMode === "limit" && (!limitPrice || parseFloat(limitPrice) <= 0)) {
      toast.error("Please enter a valid limit price");
      return;
    }

    toast.success(
      `${orderType.toUpperCase()} order for ${qty} shares of ${symbol} placed successfully!`
    );

    // Reset form
    setQuantity("10");
    setLimitPrice("");
  };

  return (
    <div className="max-w-6xl mx-auto space-y-6">
      <div>
        <h1 className="text-3xl font-bold text-white">Trade</h1>
        <p className="text-gray-300 mt-1">Place buy or sell orders</p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Order Form */}
        <div className="lg:col-span-2">
          <Card className="text-white">
            <CardHeader>
              <CardTitle className="text-white">Place Order</CardTitle>
            </CardHeader>
            <CardContent>
              <Tabs value={orderType} onValueChange={(v) => setOrderType(v as "buy" | "sell")}>
                <TabsList className="grid w-full grid-cols-2 mb-6">
                  <TabsTrigger value="buy" className="data-[state=active]:bg-green-600">
                    <TrendingUp className="w-4 h-4 mr-2" />
                    Buy
                  </TabsTrigger>
                  <TabsTrigger value="sell" className="data-[state=active]:bg-red-600">
                    <TrendingDown className="w-4 h-4 mr-2" />
                    Sell
                  </TabsTrigger>
                </TabsList>

                <TabsContent value="buy" className="space-y-4">
                  <OrderForm
                    symbol={symbol}
                    setSymbol={setSymbol}
                    quantity={quantity}
                    setQuantity={setQuantity}
                    orderMode={orderMode}
                    setOrderMode={setOrderMode}
                    limitPrice={limitPrice}
                    setLimitPrice={setLimitPrice}
                    timeInForce={timeInForce}
                    setTimeInForce={setTimeInForce}
                  />
                </TabsContent>

                <TabsContent value="sell" className="space-y-4">
                  <OrderForm
                    symbol={symbol}
                    setSymbol={setSymbol}
                    quantity={quantity}
                    setQuantity={setQuantity}
                    orderMode={orderMode}
                    setOrderMode={setOrderMode}
                    limitPrice={limitPrice}
                    setLimitPrice={setLimitPrice}
                    timeInForce={timeInForce}
                    setTimeInForce={setTimeInForce}
                  />
                </TabsContent>
              </Tabs>
            </CardContent>
          </Card>
        </div>

        {/* Order Summary */}
        <div className="space-y-6">
          <Card className="text-white">
            <CardHeader>
              <CardTitle className="text-white">Order Summary</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              {stock && (
                <>
                  <div>
                    <p className="text-sm text-gray-300">Stock</p>
                    <p className="text-xl font-bold text-white">{symbol}</p>
                    <p className="text-sm text-gray-300">{stock.name}</p>
                  </div>

                  <div className="border-t pt-4 space-y-3">
                    <div className="flex justify-between">
                      <span className="text-gray-300">Current Price</span>
                      <span className="font-semibold text-white">${stock.price.toFixed(2)}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-gray-300">Quantity</span>
                      <span className="font-semibold text-white">{qty}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-gray-300">Order Price</span>
                      <span className="font-semibold text-white">
                        ${price.toFixed(2)}
                        {orderMode === "market" && " (Market)"}
                      </span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-gray-300">Subtotal</span>
                      <span className="font-semibold text-white">
                        ${total.toFixed(2)}
                      </span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span className="text-gray-300">Est. Fee (0.1%)</span>
                      <span className="text-gray-300">${estimatedFee.toFixed(2)}</span>
                    </div>
                  </div>

                  <div className="border-t pt-4">
                    <div className="flex justify-between items-center">
                      <span className="text-lg font-semibold text-white">Total</span>
                      <span className="text-2xl font-bold text-white">
                        ${totalWithFee.toFixed(2)}
                      </span>
                    </div>
                  </div>

                  <Button
                    className={`w-full h-12 ${
                      orderType === "buy"
                        ? "bg-green-600 hover:bg-green-700"
                        : "bg-red-600 hover:bg-red-700"
                    }`}
                    onClick={handlePlaceOrder}
                  >
                    {orderType === "buy" ? "Place Buy Order" : "Place Sell Order"}
                  </Button>
                </>
              )}
            </CardContent>
          </Card>

          <Card className="text-white">
            <CardContent className="p-4">
              <div className="flex gap-2">
                <AlertCircle className="w-5 h-5 text-blue-600 flex-shrink-0 mt-0.5" />
                <div className="text-sm text-gray-300">
                  <p className="font-semibold text-white mb-1">Trading Information</p>
                  <p className="text-gray-300">Market orders execute at current market price. Limit orders execute only at your specified price or better.</p>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>

      {/* Quick Trade Buttons */}
      <Card className="text-white">
        <CardHeader>
          <CardTitle className="text-white">Quick Trade</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            {Object.keys(mockStocks).slice(0, 8).map((sym) => (
              <Button
                key={sym}
                variant="outline"
                onClick={() => setSymbol(sym)}
                className={symbol === sym ? "border-blue-600 bg-blue-50" : ""}
              >
                {sym}
              </Button>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

interface OrderFormProps {
  symbol: string;
  setSymbol: (value: string) => void;
  quantity: string;
  setQuantity: (value: string) => void;
  orderMode: "market" | "limit";
  setOrderMode: (value: "market" | "limit") => void;
  limitPrice: string;
  setLimitPrice: (value: string) => void;
  timeInForce: string;
  setTimeInForce: (value: string) => void;
}

function OrderForm({
  symbol,
  setSymbol,
  quantity,
  setQuantity,
  orderMode,
  setOrderMode,
  limitPrice,
  setLimitPrice,
  timeInForce,
  setTimeInForce,
}: OrderFormProps) {
  return (
    <>
      <div className="space-y-2">
        <Label className="text-white" htmlFor="symbol">Symbol</Label>
        <Select value={symbol} onValueChange={setSymbol}>
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {Object.keys(mockStocks).map((sym) => (
              <SelectItem key={sym} value={sym}>
                {sym} - {mockStocks[sym].name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="space-y-2">
        <Label className="text-white" htmlFor="orderMode">Order Type</Label>
        <Select value={orderMode} onValueChange={(v) => setOrderMode(v as "market" | "limit")}>
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="market">Market Order</SelectItem>
            <SelectItem value="limit">Limit Order</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <div className="space-y-2">
        <Label className="text-white" htmlFor="quantity">Quantity</Label>
        <Input
          id="quantity"
          type="number"
          value={quantity}
          onChange={(e) => setQuantity(e.target.value)}
          placeholder="Enter quantity"
          min="1"
        />
      </div>

      {orderMode === "limit" && (
        <div className="space-y-2">
          <Label className="text-white" htmlFor="limitPrice">Limit Price</Label>
          <Input
            id="limitPrice"
            type="number"
            value={limitPrice}
            onChange={(e) => setLimitPrice(e.target.value)}
            placeholder="Enter limit price"
            step="0.01"
            min="0.01"
          />
        </div>
      )}

      <div className="space-y-2">
        <Label className="text-white" htmlFor="timeInForce">Time in Force</Label>
        <Select value={timeInForce} onValueChange={setTimeInForce}>
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="day">Day</SelectItem>
            <SelectItem value="gtc">Good Till Cancelled</SelectItem>
            <SelectItem value="ioc">Immediate or Cancel</SelectItem>
          </SelectContent>
        </Select>
      </div>
    </>
  );
}
