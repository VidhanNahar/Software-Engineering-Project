import { createBrowserRouter, Navigate } from "react-router";
import { ProtectedRoute, AdminRoute } from "./components/RouteGuards";
import Login from "./pages/Login";
import Dashboard from "./pages/Dashboard";
import StockDetails from "./pages/StockDetails";
import Portfolio from "./pages/Portfolio";
import Trade from "./pages/Trade";
import MarketOverview from "./pages/MarketOverview";
import Admin from "./pages/Admin";
import Layout from "./components/Layout";

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
      {
        path: "admin",
        element: (
          <AdminRoute>
            <Admin />
          </AdminRoute>
        ),
      },
    ],
  },
  {
    path: "*",
    element: <Navigate to="/" replace />,
  },
]);
