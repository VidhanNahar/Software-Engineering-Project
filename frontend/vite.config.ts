import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

// https://vite.dev/config/
export default defineConfig({
  plugins: [tailwindcss(), react()],
  server: {
    proxy: {
      "/api/auth": {
        target: "http://localhost:8091",
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api\/auth/, "/auth"),
      },
      "/api": {
        target: "http://localhost:8091",
        changeOrigin: true,
      },
      "/ws": {
        target: "ws://localhost:8091",
        ws: true,
        changeOrigin: true,
      },
    },
  },
});
