import type { Timeframe } from "../../types/ohlcv";

type Props = {
  value: Timeframe;
  onChange: (next: Timeframe) => void;
};

const TIMEFRAMES: Timeframe[] = ["1D", "5D", "1M", "6M", "1Y"];

export function TimeframeSelector({ value, onChange }: Props) {
  return (
    <div className="flex flex-wrap items-center gap-2">
      {TIMEFRAMES.map((tf) => (
        <button
          key={tf}
          type="button"
          onClick={() => onChange(tf)}
          className={`rounded-md border px-3 py-1.5 text-xs font-semibold transition-colors ${
            value === tf
              ? "border-blue-500 bg-blue-600 text-white"
              : "border-gray-700 bg-gray-900 text-gray-300 hover:border-blue-500 hover:text-white"
          }`}
        >
          {tf}
        </button>
      ))}
    </div>
  );
}
