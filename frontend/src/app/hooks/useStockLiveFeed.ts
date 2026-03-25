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
  const intentionalCloseRef = useRef<boolean>(false);
  const reconnectTimeoutRef = useRef<number | null>(null);

  const normalizedSymbol = useMemo(() => selectedSymbol.toUpperCase(), [selectedSymbol]);

  useEffect(() => {
    setCandles(generateMockOHLCV(basePrice, timeframe));
    setPrice(basePrice);
  }, [basePrice, timeframe, normalizedSymbol]);

  useEffect(() => {
    isComponentMountedRef.current = true;
    intentionalCloseRef.current = false;

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
        intentionalCloseRef.current = false;
        const ws = new WebSocket(websocketUrl());

        ws.onopen = () => {
          if (isComponentMountedRef.current) {
            console.log("✅ Live feed WebSocket connected");
            setConnected(true);
          }
        };

        ws.onclose = (event) => {
          if (!isComponentMountedRef.current) {
            return;
          }

          setConnected(false);

          if (intentionalCloseRef.current) {
            return;
          }

          console.warn(
            `🔌 Live feed WebSocket closed (code=${event.code}${event.reason ? `, reason=${event.reason}` : ""})`,
          );

          if (reconnectTimeoutRef.current !== null) {
            window.clearTimeout(reconnectTimeoutRef.current);
          }
          reconnectTimeoutRef.current = window.setTimeout(() => {
            reconnectTimeoutRef.current = null;
            connectWebSocket();
          }, 1000);
        };

        ws.onerror = (error) => {
          if (!isComponentMountedRef.current) return;
          console.error("❌ Live feed WebSocket error:", error);
          setConnected(false);
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
      intentionalCloseRef.current = true;
      if (reconnectTimeoutRef.current !== null) {
        window.clearTimeout(reconnectTimeoutRef.current);
        reconnectTimeoutRef.current = null;
      }
      if (wsRef.current && wsRef.current.readyState !== WebSocket.CLOSED) {
        wsRef.current.close();
      }
      wsRef.current = null;
      setConnected(false);
    };
  }, [normalizedSymbol]);

  useEffect(() => {
    return () => {
      isComponentMountedRef.current = false;
      intentionalCloseRef.current = true;
      if (reconnectTimeoutRef.current !== null) {
        window.clearTimeout(reconnectTimeoutRef.current);
      }
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
