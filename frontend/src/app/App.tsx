import { RouterProvider } from "react-router";
import { router } from "./routes.tsx";
import { Toaster } from "sonner";
import { ThemeProvider } from "./context/ThemeContext";

export default function App() {
  return (
    <ThemeProvider>
      <RouterProvider router={router} />
      <Toaster position="top-right" />
    </ThemeProvider>
  );
}