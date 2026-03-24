import { useEffect, useMemo, useState, useRef } from "react";
import type { CandlePoint, LiveFeedState, Timeframe } from "../types/ohlcv";
import { applyLiveTick, generateMockOHLCV } from "../utils/mockOhlcv";

type SnapshotStock = {
  symbol: string;
  price: number;
};

function websocketUrl(): string {
  const protocol = window.location.protocol === "https:" ? "wss" : "ws";
  return `${protocol}://${window.location.host}/ws/stocks`;
}

export function useStockLiveFeed(selectedSymbol: string, basePrice: number, timeframe: Timeframe): LiveFeedState {
  const [candles, setCandles] = useState<CandlePoint[]>(() => generateMockOHLCV(basePrice, timeframe));
  const [price, setPrice] = useState<number>(basePrice);
  const [connected, setConnected] = useState<boolean>(false);
  const [lastUpdatedAt, setLastUpdatedAt] = useState<number | undefined>(undefined);
  const [marketOpen, setMarketOpen] = useState<boolean>(true);
  const wsRef = useRef<WebSocket | null>(null);
  const isComponentMountedRef = useRef<boolean>(true);

  const normalizedSymbol = useMemo(() => selectedSymbol.toUpperCase(), [selectedSymbol]);

  useEffect(() => {
    setCandles(generateMockOHLCV(basePrice, timeframe));
    setPrice(basePrice);
  }, [basePrice, timeframe, normalizedSymbol]);

  useEffect(() => {
    // Mark component as mounted
    isComponentMountedRef.current = true;

    const handleLivePrice = (incoming: number) => {
      if (isComponentMountedRef.current) {
        setPrice(incoming);
        setCandles((prev) => applyLiveTick(prev, incoming));
        setLastUpdatedAt(Date.now());
      }
    };

    const connectWebSocket = () => {
      if (!isComponentMountedRef.current) return;
      
      try {
        const ws = new WebSocket(websocketUrl());

        ws.onopen = () => {
          if (isComponentMountedRef.current) {
            console.log("✅ Live feed WebSocket connected");
            setConnected(true);
          }
        };

        ws.onclose = () => {
          if (isComponentMountedRef.current) {
            console.log("🔌 Live feed WebSocket closed");
            setConnected(false);
          }
        };

        ws.onerror = (error) => {
          if (isComponentMountedRef.current) {
            console.error("❌ Live feed WebSocket error:", error);
            setConnected(false);
          }
        };

        ws.onmessage = (event: MessageEvent<string>) => {
          if (!isComponentMountedRef.current) return;
          
          try {
            const payload = JSON.parse(event.data) as { 
              stocks?: SnapshotStock[]; 
              ticks?: SnapshotStock[];
              market_open?: boolean;
              type?: string;
            };
            
            if (typeof payload.market_open === "boolean") {
              setMarketOpen(payload.market_open);
            }
            
            // Handle both "stocks_snapshot" (stocks array) and "stock_tick" (ticks array)
            const stocksArray = payload.stocks || payload.ticks;
            
            if (!stocksArray || stocksArray.length === 0) {
              console.debug("📡 WebSocket message received but no stocks/ticks data");
              return;
            }
            
            const target = stocksArray.find((s) => {
              const symbol = (s.symbol || s.Symbol || "").toUpperCase();
              return symbol === normalizedSymbol;
            });
            
            if (!target) {
              console.debug(`📡 No data for ${normalizedSymbol} in message`);
              return;
            }
            
            const priceValue = target.price || (target as any).Price;
            
            if (typeof priceValue !== "number") {
              console.debug(`⚠️ Invalid price for ${normalizedSymbol}: ${priceValue}`);
              return;
            }
            
            if (payload.market_open === false) {
              console.debug("📡 Market closed, ignoring price update");
              return;
            }
            
            console.debug(`💰 ${normalizedSymbol}: $${priceValue}`);
            handleLivePrice(priceValue);
          } catch (e) {
            console.error("❌ Failed to parse WebSocket message:", e);
          }
        };

        wsRef.current = ws;
      } catch (error) {
        console.error("Failed to create WebSocket:", error);
        if (isComponentMountedRef.current) {
          setConnected(false);
        }
      }
    };

    connectWebSocket();

    return () => {
      if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
        wsRef.current.close();
      }
      wsRef.current = null;
    };
  }, [normalizedSymbol]);

  // Mark component as unmounted on cleanup
  useEffect(() => {
    return () => {
      isComponentMountedRef.current = false;
    };
  }, []);

  useEffect(() => {
    if (!marketOpen) return;
    setCandles(generateMockOHLCV(basePrice, timeframe));
    setPrice(basePrice);
  }, [basePrice, timeframe, marketOpen]);

  return {
    candles,
    price,
    connected,
    lastUpdatedAt,
    marketOpen,
  };
}
