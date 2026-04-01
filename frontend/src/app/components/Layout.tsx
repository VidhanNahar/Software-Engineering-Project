import { Outlet, useNavigate, useLocation } from "react-router";
import {
  LayoutDashboard,
  TrendingUp,
  Wallet,
  BarChart3,
  User,
  LogOut,
  Bell,
  Search,
  Moon,
  Sun,
  Shield,
} from "lucide-react";
import { Button } from "./ui/button";
import { Input } from "./ui/input";
import { toast } from "sonner";
import { useTheme } from "../context/ThemeContext";
import { authApi } from "../api";
import { isAdmin } from "../utils/auth";

export default function Layout() {
  const navigate = useNavigate();
  const location = useLocation();
  const { theme, toggleTheme } = useTheme();
  const userRole = localStorage.getItem("user_role") || "user";
  const userName = localStorage.getItem("user_name") || "Trader";
  const userInitials = userName
    .split(" ")
    .map((n) => n[0])
    .join("")
    .toUpperCase()
    .slice(0, 2);

  const handleLogout = async () => {
    try {
      const refreshToken = localStorage.getItem("refresh_token");
      if (refreshToken) {
        await authApi.logout(refreshToken);
      }
    } catch (error) {
      console.error("Logout API failed", error);
    } finally {
      localStorage.removeItem("isLoggedIn");
      localStorage.removeItem("access_token");
      localStorage.removeItem("refresh_token");
      localStorage.removeItem("user_id");
      localStorage.removeItem("user_role");
      localStorage.removeItem("user_name");
      toast.success("Logged out successfully");
      navigate("/login");
    }
  };

  const allNavItems = [
    { path: "/", icon: LayoutDashboard, label: "Dashboard" },
    { path: "/market", icon: TrendingUp, label: "Market" },
    { path: "/portfolio", icon: Wallet, label: "Portfolio" },
    { path: "/trade", icon: BarChart3, label: "Trade" },
    ...(isAdmin() ? [{ path: "/admin", icon: Shield, label: "Admin" }] : []),
  ];

  return (
    <div className="flex h-screen bg-background">
      {/* Sidebar */}
      <aside className="w-64 bg-card border-r border-border flex flex-col">
        <div className="p-6 border-b border-border">
          <h1 className="text-2xl font-bold text-blue-600">FinXGrow</h1>
          <p className="text-sm text-muted-foreground">Professional Trading</p>
        </div>

        <nav className="flex-1 p-4">
          <div className="space-y-2">
            {allNavItems.map((item) => {
              const Icon = item.icon;
              const isActive = location.pathname === item.path;
              return (
                <button
                  key={item.path}
                  onClick={() => navigate(item.path)}
                  className={`w-full flex items-center gap-3 px-4 py-3 rounded-lg transition-colors ${
                    isActive
                      ? "bg-blue-600 text-white"
                      : "text-foreground hover:bg-accent hover:text-accent-foreground"
                  }`}
                >
                  <Icon className="w-5 h-5" />
                  <span>{item.label}</span>
                </button>
              );
            })}
          </div>
        </nav>

        <div className="p-4 border-t border-border space-y-2">
          <Button
            variant="ghost"
            className="w-full justify-start"
            onClick={() => navigate("/profile")}
          >
            <User className="w-5 h-5 mr-3" />
            Profile
          </Button>
          <Button
            variant="ghost"
            className="w-full justify-start text-red-600 hover:text-red-700 hover:bg-red-50 dark:hover:bg-red-950"
            onClick={handleLogout}
          >
            <LogOut className="w-5 h-5 mr-3" />
            Logout
          </Button>
        </div>
      </aside>

      {/* Main Content */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {/* Header */}
        <header className="bg-card border-b border-border p-4">
          <div className="flex items-center justify-between">
            <div className="flex-1 max-w-xl">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-5 h-5 text-muted-foreground" />
                <Input
                  placeholder="Search stocks, symbols..."
                  className="pl-10"
                />
              </div>
            </div>

            <div className="flex items-center gap-4">
              <Button
                variant="ghost"
                size="icon"
                onClick={toggleTheme}
                title={
                  theme === "light"
                    ? "Switch to dark mode"
                    : "Switch to light mode"
                }
              >
                {theme === "light" ? (
                  <Moon className="w-5 h-5" />
                ) : (
                  <Sun className="w-5 h-5" />
                )}
              </Button>
              <button
                onClick={() => navigate("/profile")}
                className="flex items-center gap-3 text-left hover:opacity-80 transition-opacity"
              >
                <div className="text-right">
                  <p className="text-sm font-medium text-foreground">
                    {userName}
                  </p>
                  <p className="text-xs text-muted-foreground capitalize">
                    {userRole === "admin" ? "Administrator" : "Trader Account"}
                  </p>
                </div>
                <div className="w-10 h-10 rounded-full bg-blue-600 flex items-center justify-center text-white font-semibold shrink-0">
                  {userInitials}
                </div>
              </button>
            </div>
          </div>
        </header>

        {/* Page Content */}
        <main className="flex-1 overflow-auto p-6 bg-background">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
