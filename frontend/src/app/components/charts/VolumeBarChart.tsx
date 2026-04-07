import {
  BarElement,
  CategoryScale,
  Chart as ChartJS,
  Legend,
  LinearScale,
  Tooltip,
} from "chart.js";
import { Bar } from "react-chartjs-2";
import { useTheme } from "../../context/ThemeContext";

ChartJS.register(CategoryScale, LinearScale, BarElement, Tooltip, Legend);

type VolumePoint = {
  label: string;
  value: number;
  isUp: boolean;
};

type Props = {
  data: VolumePoint[];
};

export function VolumeBarChart({ data }: Props) {
  const { theme } = useTheme();
  const isDark = theme === "dark";
  
  const textColor = isDark ? "#94a3b8" : "#64748b";
  const gridColor = isDark ? "rgba(51, 65, 85, 0.25)" : "rgba(0, 0, 0, 0.05)";

  const chartData = {
    labels: data.map((d) => d.label),
    datasets: [
      {
        label: "Volume",
        data: data.map((d) => d.value),
        backgroundColor: data.map((d) => (d.isUp ? "rgba(16, 185, 129, 0.65)" : "rgba(239, 68, 68, 0.65)")),
        borderWidth: 0,
      },
    ],
  };

  return (
    <Bar
      data={chartData}
      options={{
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
          legend: { display: false },
          tooltip: {
            callbacks: {
              label: (ctx) => `${Number(ctx.parsed.y).toLocaleString()} shares`,
            },
          },
        },
        scales: {
          x: { 
            ticks: { color: textColor, maxTicksLimit: 7 }, 
            grid: { display: false } 
          },
          y: {
            ticks: {
              color: textColor,
              callback: (value) => Number(value).toLocaleString(),
            },
            grid: { color: gridColor },
          },
        },
      }}
    />
  );
}
