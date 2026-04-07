import {
  CategoryScale,
  Chart as ChartJS,
  Filler,
  Legend,
  LineElement,
  LinearScale,
  PointElement,
  Tooltip,
} from "chart.js";
import { Line } from "react-chartjs-2";
import { formatPrice } from "../../utils/currency";
import { useTheme } from "../../context/ThemeContext";

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Tooltip, Legend, Filler);

type DataPoint = {
  label: string;
  value: number;
};

type Props = {
  data: DataPoint[];
  symbol: string;
};

export function ClosePriceLineChart({ data, symbol }: Props) {
  const { theme } = useTheme();
  const isDark = theme === "dark";
  
  const textColor = isDark ? "#94a3b8" : "#64748b";
  const gridColor = isDark ? "rgba(51, 65, 85, 0.25)" : "rgba(0, 0, 0, 0.05)";

  const chartData = {
    labels: data.map((d) => d.label),
    datasets: [
      {
        label: `${symbol} Close`,
        data: data.map((d) => d.value),
        borderColor: "#3b82f6",
        backgroundColor: "rgba(59, 130, 246, 0.18)",
        borderWidth: 2,
        pointRadius: 0,
        tension: 0.2,
        fill: true,
      },
    ],
  };

  return (
    <Line
      data={chartData}
      options={{
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
          legend: { display: false },
          tooltip: {
            mode: "index",
            intersect: false,
            callbacks: {
              label: (ctx) => formatPrice(Number(ctx.parsed.y)),
            },
          },
        },
        scales: {
          x: { 
            ticks: { color: textColor, maxTicksLimit: 7 }, 
            grid: { color: gridColor } 
          },
          y: {
            ticks: {
              color: textColor,
              callback: (value) => formatPrice(Number(value)),
            },
            grid: { color: gridColor },
          },
        },
      }}
    />
  );
}
