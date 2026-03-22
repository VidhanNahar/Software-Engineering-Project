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
              label: (ctx) => `$${Number(ctx.parsed.y).toFixed(2)}`,
            },
          },
        },
        scales: {
          x: { ticks: { color: "#94a3b8", maxTicksLimit: 7 }, grid: { color: "rgba(51, 65, 85, 0.25)" } },
          y: {
            ticks: {
              color: "#94a3b8",
              callback: (value) => `$${value}`,
            },
            grid: { color: "rgba(51, 65, 85, 0.25)" },
          },
        },
      }}
    />
  );
}
