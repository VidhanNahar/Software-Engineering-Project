import type { StockOption } from "../../types/ohlcv";

type Props = {
  value: string;
  options: StockOption[];
  onChange: (symbol: string) => void;
};

export function StockSelector({ value, options, onChange }: Props) {
  return (
    <div className="flex items-center gap-2">
      <label htmlFor="stock-selector" className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
        Stock
      </label>
      <select
        id="stock-selector"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="rounded-md border border-border bg-card px-3 py-1.5 text-sm text-foreground outline-none focus:border-blue-500"
      >
        {options.map((opt) => (
          <option key={opt.stockId} value={opt.symbol} className="bg-card text-foreground">
            {opt.symbol} - {opt.name}
          </option>
        ))}
      </select>
    </div>
  );
}
