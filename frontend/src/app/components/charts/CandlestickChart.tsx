import { useEffect, useRef } from "react";
import {
  CandlestickSeries,
  ColorType,
  createChart,
  type CandlestickData,
  type IChartApi,
  type UTCTimestamp,
} from "lightweight-charts";
import type { CandlePoint } from "../../types/ohlcv";
import { useTheme } from "../../context/ThemeContext";

type Props = {
  data: CandlePoint[];
};

export function CandlestickChart({ data }: Props) {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const chartRef = useRef<IChartApi | null>(null);
  const seriesRef = useRef<any>(null);
  const { theme } = useTheme();

  useEffect(() => {
    if (!containerRef.current) return;

    const isDark = theme === "dark";
    const backgroundColor = isDark ? "#111827" : "#ffffff";
    const textColor = isDark ? "#cbd5e1" : "#1f2937";
    const gridColor = isDark ? "rgba(148, 163, 184, 0.1)" : "rgba(0, 0, 0, 0.05)";

    const chart = createChart(containerRef.current, {
      autoSize: true,
      layout: {
        background: { type: ColorType.Solid, color: backgroundColor },
        textColor: textColor,
      },
      grid: {
        vertLines: { color: gridColor },
        horzLines: { color: gridColor },
      },
      rightPriceScale: { 
        borderColor: isDark ? "rgba(148, 163, 184, 0.35)" : "rgba(0, 0, 0, 0.1)",
      },
      timeScale: {
        borderColor: isDark ? "rgba(148, 163, 184, 0.35)" : "rgba(0, 0, 0, 0.1)",
        timeVisible: true,
      },
      crosshair: {
        vertLine: { color: "#60a5fa", width: 1 },
        horzLine: { color: "#60a5fa", width: 1 },
      },
    });

    const series = chart.addSeries(CandlestickSeries, {
      upColor: "#10b981",
      downColor: "#ef4444",
      borderVisible: false,
      wickUpColor: "#10b981",
      wickDownColor: "#ef4444",
    });

    chartRef.current = chart;
    seriesRef.current = series;

    return () => {
      chart.remove();
      chartRef.current = null;
      seriesRef.current = null;
    };
  }, [theme]); // Re-create chart on theme change to update colors properly

  useEffect(() => {
    if (!seriesRef.current || data.length === 0) return;

    const candleData: CandlestickData[] = data.map((d) => ({
      time: d.time as UTCTimestamp,
      open: d.open,
      high: d.high,
      low: d.low,
      close: d.close,
    }));

    seriesRef.current.setData(candleData);
    chartRef.current?.timeScale().fitContent();
  }, [data]);

  return <div ref={containerRef} className="h-[360px] w-full rounded-md" />;
}
