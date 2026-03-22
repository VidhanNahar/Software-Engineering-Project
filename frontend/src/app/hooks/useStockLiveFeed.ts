import { useEffect, useMemo, useState } from "react";
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

  const normalizedSymbol = useMemo(() => selectedSymbol.toUpperCase(), [selectedSymbol]);

  useEffect(() => {
    setCandles(generateMockOHLCV(basePrice, timeframe));
    setPrice(basePrice);
  }, [basePrice, timeframe, normalizedSymbol]);

  useEffect(() => {
    let ws: WebSocket | null = null;

    const handleLivePrice = (incoming: number) => {
      setPrice(incoming);
      setCandles((prev) => applyLiveTick(prev, incoming));
      setLastUpdatedAt(Date.now());
    };

    try {
      ws = new WebSocket(websocketUrl());

      ws.onopen = () => {
        setConnected(true);
      };

      ws.onclose = () => {
        setConnected(false);
      };

      ws.onerror = () => {
        setConnected(false);
      };

      ws.onmessage = (event: MessageEvent<string>) => {
        try {
          const payload = JSON.parse(event.data) as { stocks?: SnapshotStock[]; market_open?: boolean };
          if (typeof payload.market_open === "boolean") {
            setMarketOpen(payload.market_open);
          }
          if (!payload?.stocks || payload.stocks.length === 0) return;
          const target = payload.stocks.find((s) => s.symbol?.toUpperCase() === normalizedSymbol);
          if (!target || typeof target.price !== "number") return;
          if (payload.market_open === false) return;
          handleLivePrice(target.price);
        } catch {
          // Ignore malformed websocket payloads.
        }
      };
    } catch {
      setConnected(false);
    }

    return () => {
      if (ws) ws.close();
    };
  }, [normalizedSymbol]);

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
  };
}
