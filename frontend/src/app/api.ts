const BASE_URL = import.meta.env.VITE_API_URL || "http://20.193.252.172:8080";

export const API_URL = `${BASE_URL}/api`;
export const AUTH_API_URL = `${BASE_URL}/auth`;
export const getAuthHeaders = () => {
  const token = localStorage.getItem("access_token");
  return {
    "Content-Type": "application/json",
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
  };
};

export const request = async (
  baseUrl: string,
  endpoint: string,
  options: RequestInit = {},
) => {
  const response = await fetch(`${baseUrl}${endpoint}`, {
    ...options,
    headers: {
      ...getAuthHeaders(),
      ...options.headers,
    },
  });

  if (!response.ok) {
    let errorMsg = "Something went wrong";
    try {
      const data = await response.json();
      errorMsg = data.error || data.message || errorMsg;
    } catch (e) {
      try {
        const text = await response.text();
        if (text) errorMsg = text;
      } catch {
        // Ignored
      }
    }

    if (response.status === 401) {
      localStorage.removeItem("isLoggedIn");
      localStorage.removeItem("access_token");
      localStorage.removeItem("refresh_token");
      localStorage.removeItem("user_id");
      if (window.location.pathname !== "/login") {
        window.location.href = "/login";
      }
    }

    throw new Error(errorMsg);
  }

  if (response.status === 204) {
    return null;
  }

  return response.json();
};

export const apiCall = async (endpoint: string, options: RequestInit = {}) =>
  request(API_URL, endpoint, options);

export const authApi = {
  login: (data: any) =>
    request(AUTH_API_URL, "/login", {
      method: "POST",
      body: JSON.stringify(data),
    }),
  register: (data: any) =>
    request(AUTH_API_URL, "/register", {
      method: "POST",
      body: JSON.stringify(data),
    }),
  logout: (refreshToken: string) =>
    request(AUTH_API_URL, "/logout", {
      method: "POST",
      body: JSON.stringify({ refresh_token: refreshToken }),
    }),
  refresh: (refreshToken: string) =>
    request(AUTH_API_URL, "/refresh", {
      method: "POST",
      body: JSON.stringify({ refresh_token: refreshToken }),
    }),
  verify: (data: { email_id: string; otp: string }) =>
    request(AUTH_API_URL, "/verify", {
      method: "POST",
      body: JSON.stringify(data),
    }),
  forgotPassword: (email: string) =>
    request(AUTH_API_URL, "/forgot-password", {
      method: "POST",
      body: JSON.stringify({ email_id: email }),
    }),
  resetPassword: (data: any) =>
    request(AUTH_API_URL, "/reset-password", {
      method: "POST",
      body: JSON.stringify(data),
    }),
};

export const stockApi = {
  getAll: () => apiCall("/stocks"),
  search: (query: string) =>
    apiCall(`/stocks/search?q=${encodeURIComponent(query)}`),
  getById: (id: string) => apiCall(`/stocks/${id}`),
  getBySymbol: (symbol: string) =>
    apiCall(`/stocks/symbol/${encodeURIComponent(symbol)}`),
  getStats: (id: string) => apiCall(`/stocks/${id}/stats`),
  getTicks: (symbol: string, limit = 200) =>
    apiCall(
      `/stocks/symbol/${encodeURIComponent(symbol)}/ticks?limit=${limit}`,
    ),
  getCandles: (symbol: string, timeframe = "1m", limit = 200) =>
    apiCall(
      `/stocks/symbol/${encodeURIComponent(symbol)}/candles?timeframe=${encodeURIComponent(
        timeframe,
      )}&limit=${limit}`,
    ),
};

export const transactionApi = {
  buy: (data: { stock_id: string; quantity: number }) =>
    apiCall("/transactions/buy", {
      method: "POST",
      body: JSON.stringify(data),
    }),
  sell: (data: { stock_id: string; quantity: number }) =>
    apiCall("/transactions/sell", {
      method: "POST",
      body: JSON.stringify(data),
    }),
  getHistory: () => apiCall("/transactions/history"),
};

export const portfolioApi = {
  get: () => apiCall("/portfolio"),
};

export const walletApi = {
  get: () => apiCall("/wallet"),
};

export const watchlistApi = {
  get: () => apiCall("/watchlist"),
  add: (data: { stock_id: string; watchlist_name?: string }) =>
    apiCall("/watchlist", {
      method: "POST",
      body: JSON.stringify(data),
    }),
  remove: (watchlistId: string) =>
    apiCall(`/watchlist/${watchlistId}`, {
      method: "DELETE",
    }),
};

export const adminApi = {
  createStock: (data: any) =>
    apiCall("/admin/stocks", {
      method: "POST",
      body: JSON.stringify(data),
    }),
  updateStock: (stockId: string, data: any) =>
    apiCall(`/admin/stocks/${stockId}`, {
      method: "PUT",
      body: JSON.stringify(data),
    }),
  deleteStock: (stockId: string) =>
    apiCall(`/admin/stocks/${stockId}`, {
      method: "DELETE",
    }),
  getTopStocks: () => apiCall("/admin/stocks/top"),
  getMarketStatus: () =>
    apiCall(`/market/status?ts=${Date.now()}`, {
      method: "GET",
      cache: "no-store",
    }),
  startMarket: () =>
    apiCall("/admin/market/start", {
      method: "POST",
    }),
  stopMarket: () =>
    apiCall("/admin/market/stop", {
      method: "POST",
    }),
};

export const userApi = {
  getProfile: (userId: string) => apiCall(`/user/${userId}`),
  completeKyc: (data: { aadhar_id?: string; pan_id?: string }) =>
    apiCall("/user/kyc/complete", {
      method: "POST",
      body: JSON.stringify(data),
    }),
};
