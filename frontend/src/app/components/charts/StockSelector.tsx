import type { StockOption } from "../../types/ohlcv";

type Props = {
  value: string;
  options: StockOption[];
  onChange: (symbol: string) => void;
};

export function StockSelector({ value, options, onChange }: Props) {
  return (
    <div className="flex items-center gap-2">
      <label htmlFor="stock-selector" className="text-xs font-semibold uppercase tracking-wide text-gray-400">
        Stock
      </label>
      <select
        id="stock-selector"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="rounded-md border border-gray-700 bg-gray-900 px-3 py-1.5 text-sm text-white outline-none focus:border-blue-500"
      >
        {options.map((opt) => (
          <option key={opt.stockId} value={opt.symbol}>
            {opt.symbol} - {opt.name}
          </option>
        ))}
      </select>
    </div>
  );
}
