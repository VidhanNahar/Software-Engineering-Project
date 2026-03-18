export const API_URL = "/api";

export const getAuthHeaders = () => {
  const token = localStorage.getItem("access_token");
  return {
    "Content-Type": "application/json",
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
  };
};

export const apiCall = async (endpoint: string, options: RequestInit = {}) => {
  const response = await fetch(`${API_URL}${endpoint}`, {
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
      errorMsg = data.error || errorMsg;
    } catch (e) {
      // Ignored
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

export const authApi = {
  login: (data: any) =>
    apiCall("/auth/login", {
      method: "POST",
      body: JSON.stringify(data),
    }),
  register: (data: any) =>
    apiCall("/auth/register", {
      method: "POST",
      body: JSON.stringify(data),
    }),
  logout: (refreshToken: string) =>
    apiCall("/auth/logout", {
      method: "POST",
      body: JSON.stringify({ refresh_token: refreshToken }),
    }),
  refresh: (refreshToken: string) =>
    apiCall("/auth/refresh", {
      method: "POST",
      body: JSON.stringify({ refresh_token: refreshToken }),
    }),
};

export const stockApi = {
  getAll: () => apiCall("/stocks"),
  search: (query: string) =>
    apiCall(`/stocks/search?q=${encodeURIComponent(query)}`),
  getById: (id: string) => apiCall(`/stocks/${id}`),
  getStats: (id: string) => apiCall(`/stocks/${id}/stats`),
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
