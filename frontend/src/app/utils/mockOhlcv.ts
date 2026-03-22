import type { CandlePoint, Timeframe } from "../types/ohlcv";

const SECONDS_PER_MINUTE = 60;

function timeframeConfig(timeframe: Timeframe): { points: number; intervalSec: number; drift: number } {
  switch (timeframe) {
    case "1D":
      return { points: 78, intervalSec: 5 * SECONDS_PER_MINUTE, drift: 0.004 };
    case "5D":
      return { points: 120, intervalSec: 30 * SECONDS_PER_MINUTE, drift: 0.006 };
    case "1M":
      return { points: 120, intervalSec: 6 * 60 * SECONDS_PER_MINUTE, drift: 0.01 };
    case "6M":
      return { points: 140, intervalSec: 24 * 60 * 60, drift: 0.015 };
    case "1Y":
      return { points: 180, intervalSec: 24 * 60 * 60, drift: 0.02 };
    default:
      return { points: 78, intervalSec: 5 * SECONDS_PER_MINUTE, drift: 0.004 };
  }
}

export function generateMockOHLCV(basePrice: number, timeframe: Timeframe): CandlePoint[] {
  const cfg = timeframeConfig(timeframe);
  const nowSec = Math.floor(Date.now() / 1000);
  const candles: CandlePoint[] = [];

  let prevClose = Math.max(basePrice * (0.96 + Math.random() * 0.05), 1);

  for (let i = cfg.points - 1; i >= 0; i -= 1) {
    const t = nowSec - i * cfg.intervalSec;
    const open = prevClose;
    const deltaPct = (Math.random() - 0.48) * cfg.drift;
    const close = Math.max(open * (1 + deltaPct), 0.5);

    const wickRange = Math.max(open, close) * (Math.random() * cfg.drift * 0.5);
    const high = Math.max(open, close) + wickRange;
    const low = Math.max(0.1, Math.min(open, close) - wickRange);

    const volumeBase = 50000 + Math.random() * 350000;
    const momentumBoost = Math.abs(close - open) * 12000;
    const volume = Math.round(volumeBase + momentumBoost);

    candles.push({
      time: t,
      open: round2(open),
      high: round2(high),
      low: round2(low),
      close: round2(close),
      volume,
    });

    prevClose = close;
  }

  return candles;
}

export function applyLiveTick(candles: CandlePoint[], livePrice: number): CandlePoint[] {
  if (candles.length === 0) return candles;

  const next = [...candles];
  const last = { ...next[next.length - 1] };
  const clamped = Math.max(0.1, livePrice);

  last.close = round2(clamped);
  last.high = round2(Math.max(last.high, clamped));
  last.low = round2(Math.min(last.low, clamped));
  last.volume = Math.round(last.volume + 150 + Math.random() * 800);

  next[next.length - 1] = last;
  return next;
}

export function toLineSeries(candles: CandlePoint[]): { label: string; value: number }[] {
  return candles.map((c) => ({
    label: formatTime(c.time),
    value: c.close,
  }));
}

export function toVolumeSeries(candles: CandlePoint[]): { label: string; value: number; isUp: boolean }[] {
  return candles.map((c) => ({
    label: formatTime(c.time),
    value: c.volume,
    isUp: c.close >= c.open,
  }));
}

function formatTime(unixSec: number): string {
  const d = new Date(unixSec * 1000);
  return `${String(d.getHours()).padStart(2, "0")}:${String(d.getMinutes()).padStart(2, "0")}`;
}

function round2(v: number): number {
  return Math.round(v * 100) / 100;
}
