import { createBrowserRouter, Navigate } from "react-router";
import Login from "./pages/Login";
import Dashboard from "./pages/Dashboard";
import StockDetails from "./pages/StockDetails";
import Portfolio from "./pages/Portfolio";
import Trade from "./pages/Trade";
import MarketOverview from "./pages/MarketOverview";
import Layout from "./components/Layout";

// Simple auth check (in real app, this would check actual auth state)
const isAuthenticated = () => {
  return localStorage.getItem("isLoggedIn") === "true";
};

const ProtectedRoute = ({ children }: { children: React.ReactNode }) => {
  return isAuthenticated() ? children : <Navigate to="/login" replace />;
};

export const router = createBrowserRouter([
  {
    path: "/login",
    Component: Login,
  },
  {
    path: "/",
    element: (
      <ProtectedRoute>
        <Layout />
      </ProtectedRoute>
    ),
    children: [
      { index: true, Component: Dashboard },
      { path: "stock/:symbol", Component: StockDetails },
      { path: "portfolio", Component: Portfolio },
      { path: "trade", Component: Trade },
      { path: "market", Component: MarketOverview },
    ],
  },
  {
    path: "*",
    element: <Navigate to="/" replace />,
  },
]);
