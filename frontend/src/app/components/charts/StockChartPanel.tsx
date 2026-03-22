import { useMemo, useState } from "react";
import type { StockOption, Timeframe } from "../../types/ohlcv";
import { TimeframeSelector } from "./TimeframeSelector";
import { StockSelector } from "./StockSelector";
import { ClosePriceLineChart } from "./ClosePriceLineChart";
import { CandlestickChart } from "./CandlestickChart";
import { VolumeBarChart } from "./VolumeBarChart";
import { toLineSeries, toVolumeSeries } from "../../utils/mockOhlcv";
import { useStockLiveFeed } from "../../hooks/useStockLiveFeed";

type Props = {
  options: StockOption[];
  selectedSymbol: string;
  onSymbolChange: (symbol: string) => void;
  basePrice: number;
};

export function StockChartPanel({ options, selectedSymbol, onSymbolChange, basePrice }: Props) {
  const [timeframe, setTimeframe] = useState<Timeframe>("1D");
  const { candles, price, connected, lastUpdatedAt } = useStockLiveFeed(selectedSymbol, basePrice, timeframe);

  const closeSeries = useMemo(() => toLineSeries(candles), [candles]);
  const volumeSeries = useMemo(() => toVolumeSeries(candles), [candles]);

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-3 rounded-lg border border-gray-800 bg-gray-950/65 p-3 md:flex-row md:items-center md:justify-between">
        <StockSelector value={selectedSymbol} options={options} onChange={onSymbolChange} />
        <TimeframeSelector value={timeframe} onChange={setTimeframe} />
        <div className="ml-auto flex items-center gap-3 text-sm">
          <div className={`h-2.5 w-2.5 rounded-full ${connected ? "bg-emerald-500" : "bg-amber-400"}`} />
          <span className="text-gray-300">{connected ? "Live via WebSocket" : "Waiting for live feed"}</span>
          <span className="font-semibold text-white">${price.toFixed(2)}</span>
          {lastUpdatedAt ? (
            <span className="text-xs text-gray-400">{new Date(lastUpdatedAt).toLocaleTimeString()}</span>
          ) : null}
        </div>
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <div className="rounded-lg border border-gray-800 bg-gray-900/60 p-3">
          <h3 className="mb-2 text-sm font-semibold text-gray-200">Candlestick (OHLC)</h3>
          <CandlestickChart data={candles} />
        </div>
        <div className="rounded-lg border border-gray-800 bg-gray-900/60 p-3">
          <h3 className="mb-2 text-sm font-semibold text-gray-200">Close Price Line</h3>
          <div className="h-[360px]">
            <ClosePriceLineChart data={closeSeries} symbol={selectedSymbol} />
          </div>
        </div>
      </div>

      <div className="rounded-lg border border-gray-800 bg-gray-900/60 p-3">
        <h3 className="mb-2 text-sm font-semibold text-gray-200">Volume</h3>
        <div className="h-[220px]">
          <VolumeBarChart data={volumeSeries} />
        </div>
      </div>
    </div>
  );
}
