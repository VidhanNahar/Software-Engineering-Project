import { Navigate } from "react-router";
import { isAuthenticated, isAdmin } from "../utils/auth";

export const ProtectedRoute = ({ children }: { children: React.ReactNode }) => {
  return isAuthenticated() ? children : <Navigate to="/login" replace />;
};

export const AdminRoute = ({ children }: { children: React.ReactNode }) => {
  if (!isAuthenticated()) return <Navigate to="/login" replace />;
  if (!isAdmin()) return <Navigate to="/" replace />;
  return children;
};
