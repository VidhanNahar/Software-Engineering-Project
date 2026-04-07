export const isAuthenticated = () => {
  const isLoggedIn = localStorage.getItem("isLoggedIn") === "true";
  const hasToken = !!localStorage.getItem("access_token");
  return isLoggedIn && hasToken;
};

export const isAdmin = () => {
  return localStorage.getItem("user_role") === "admin";
};

export const isKycVerified = () => {
  return localStorage.getItem("is_kyc_verified") === "true";
};
