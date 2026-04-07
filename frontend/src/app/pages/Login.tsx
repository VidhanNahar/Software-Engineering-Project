import { useState } from "react";
import { useNavigate } from "react-router";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "../components/ui/card";
import { TrendingUp, Eye, EyeOff, Moon, Sun, Loader2, ArrowLeft } from "lucide-react";
import { toast } from "sonner";
import { useTheme } from "../context/ThemeContext";
import { authApi } from "../api";

export default function Login() {
  const navigate = useNavigate();
  const { theme, toggleTheme } = useTheme();

  const [isRegistering, setIsRegistering] = useState(false);
  const [isVerifying, setIsVerifying] = useState(false);
  const [isForgotPassword, setIsForgotPassword] = useState(false);
  const [isResettingPassword, setIsResettingPassword] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [showPassword, setShowPassword] = useState(false);

  // Form fields
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [dateOfBirth, setDateOfBirth] = useState("");
  const [otp, setOtp] = useState("");

  // ── Email / Password ────────────────────────────────────────────────────────
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (isVerifying) {
      if (!otp) {
        toast.error("Please enter the OTP");
        return;
      }
    } else if (isResettingPassword) {
      if (!otp || !password) {
        toast.error("Please enter OTP and new password");
        return;
      }
    } else if (isForgotPassword) {
      if (!email) {
        toast.error("Please enter your email");
        return;
      }
    } else if (isRegistering) {
      if (!name || !email || !password || !dateOfBirth) {
        toast.error("Please fill in all required fields");
        return;
      }
    } else {
      if (!email || !password) {
        toast.error("Please enter email and password");
        return;
      }
    }

    setIsLoading(true);

    try {
      if (isVerifying) {
        await authApi.verify({
          email_id: email,
          otp,
        });

        toast.success("Email verified successfully! Please log in.");
        setIsVerifying(false);
        setIsRegistering(false);
        setOtp("");
        setPassword("");
      } else if (isResettingPassword) {
        await authApi.resetPassword({
          email_id: email,
          otp,
          new_password: password,
        });
        toast.success("Password reset successful! Please log in.");
        setIsResettingPassword(false);
        setIsForgotPassword(false);
        setOtp("");
        setPassword("");
      } else if (isForgotPassword) {
        await authApi.forgotPassword(email);
        toast.success("If the email exists, an OTP has been sent.");
        setIsResettingPassword(true);
      } else if (isRegistering) {
        await authApi.register({
          name,
          email_id: email,
          password,
          date_of_birth: dateOfBirth
            ? new Date(dateOfBirth).toISOString()
            : undefined,
          is_verified_email: false,
        });

        toast.success(
          "Registration successful! Please check your email for the OTP.",
        );
        setIsVerifying(true);
      } else {
        const response = await authApi.login({
          email_id: email,
          password,
        });

        localStorage.setItem("isLoggedIn", "true");
        localStorage.setItem(
          "access_token",
          response.access_token || response.token,
        );
        if (response.refresh_token) {
          localStorage.setItem("refresh_token", response.refresh_token);
        }
        if (response.user_id) {
          localStorage.setItem("user_id", response.user_id);
        }
        if (response.role) {
          localStorage.setItem("user_role", response.role);
        }
        if (response.name) {
          localStorage.setItem("user_name", response.name);
        }

        toast.success("Login successful!");
        navigate("/");
      }
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : "Authentication failed");
    } finally {
      setIsLoading(false);
    }
  };

  const toggleMode = () => {
    setIsRegistering(!isRegistering);
    setIsVerifying(false);
    setIsForgotPassword(false);
    setIsResettingPassword(false);
    setName("");
    setEmail("");
    setPassword("");
    setDateOfBirth("");
    setOtp("");
  };

  const handleBackToLogin = () => {
    setIsForgotPassword(false);
    setIsResettingPassword(false);
    setIsRegistering(false);
    setIsVerifying(false);
    setPassword("");
    setOtp("");
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 via-white to-purple-50 dark:from-gray-900 dark:via-gray-900 dark:to-gray-800 flex items-center justify-center p-4">
      <Button
        variant="ghost"
        size="icon"
        onClick={toggleTheme}
        className="absolute top-4 right-4 text-gray-600 dark:text-gray-300"
        title={
          theme === "light" ? "Switch to dark mode" : "Switch to light mode"
        }
      >
        {theme === "light" ? (
          <Moon className="w-5 h-5" />
        ) : (
          <Sun className="w-5 h-5" />
        )}
      </Button>

      <div className="w-full max-w-md">
        <Card className="w-full bg-white dark:bg-gray-900 border-gray-200 dark:border-gray-800 shadow-xl">
          <CardHeader className="text-center space-y-4">
            {(isForgotPassword || isVerifying || isResettingPassword) && (
              <button 
                onClick={handleBackToLogin}
                className="absolute left-6 top-8 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 flex items-center gap-1 text-sm"
              >
                <ArrowLeft className="w-4 h-4" /> Back
              </button>
            )}
            <div className="flex items-center justify-center gap-3">
              <div className="w-12 h-12 bg-blue-100 dark:bg-blue-900/50 rounded-xl flex items-center justify-center">
                <TrendingUp className="w-7 h-7 text-blue-600 dark:text-blue-400" />
              </div>
            </div>
            <div>
              <CardTitle className="text-2xl text-gray-900 dark:text-white">
                {isResettingPassword 
                  ? "Reset Password" 
                  : isForgotPassword 
                    ? "Forgot Password" 
                    : isVerifying
                      ? "Verify Email"
                      : isRegistering
                        ? "Create an Account"
                        : "Welcome to FinXGrow"}
              </CardTitle>
              <CardDescription className="text-gray-500 dark:text-gray-400">
                {isResettingPassword
                  ? "Enter the OTP and your new password"
                  : isForgotPassword
                    ? "Enter your email to receive an OTP"
                    : isVerifying
                      ? "Enter the OTP sent to your email"
                      : isRegistering
                        ? "Sign up to start trading today"
                        : "Sign in to your trading account to continue"}
              </CardDescription>
            </div>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit} className="space-y-4">
              {(isVerifying || isResettingPassword) && (
                <div className="space-y-2">
                  <Label
                    className="text-gray-700 dark:text-gray-200"
                    htmlFor="otp"
                  >
                    One-Time Password (OTP)
                  </Label>
                  <Input
                    id="otp"
                    type="text"
                    placeholder="Enter 6-digit OTP"
                    value={otp}
                    onChange={(e) => setOtp(e.target.value)}
                    className="bg-gray-50 dark:bg-gray-800 border-gray-300 dark:border-gray-700 text-gray-900 dark:text-white text-center tracking-[0.5em] text-lg font-mono"
                    maxLength={6}
                  />
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    An OTP has been sent to {email}
                  </p>
                </div>
              )}

              {!isVerifying && (!isResettingPassword || isForgotPassword) && (
                <>
                  {isRegistering && (
                    <div className="space-y-2">
                      <Label
                        className="text-gray-700 dark:text-gray-200"
                        htmlFor="name"
                      >
                        Full Name
                      </Label>
                      <Input
                        id="name"
                        type="text"
                        placeholder="John Doe"
                        value={name}
                        onChange={(e) => setName(e.target.value)}
                        className="bg-gray-50 dark:bg-gray-800 border-gray-300 dark:border-gray-700 text-gray-900 dark:text-white"
                      />
                    </div>
                  )}

                  <div className="space-y-2">
                    <Label
                      className="text-gray-700 dark:text-gray-200"
                      htmlFor="email"
                    >
                      Email Address
                    </Label>
                    <Input
                      id="email"
                      type="email"
                      placeholder="trader@example.com"
                      value={email}
                      onChange={(e) => setEmail(e.target.value)}
                      className="bg-gray-50 dark:bg-gray-800 border-gray-300 dark:border-gray-700 text-gray-900 dark:text-white"
                      disabled={isVerifying || isResettingPassword}
                    />
                  </div>

                  {isRegistering && (
                    <div className="space-y-2">
                      <Label
                        className="text-gray-700 dark:text-gray-200"
                        htmlFor="dob"
                      >
                        Date of Birth
                      </Label>
                      <Input
                        id="dob"
                        type="date"
                        value={dateOfBirth}
                        onChange={(e) => setDateOfBirth(e.target.value)}
                        className="bg-gray-50 dark:bg-gray-800 border-gray-300 dark:border-gray-700 text-gray-900 dark:text-white dark:[&::-webkit-calendar-picker-indicator]:filter dark:[&::-webkit-calendar-picker-indicator]:invert"
                      />
                    </div>
                  )}
                </>
              )}

              {(isRegistering || !isVerifying && !isForgotPassword || isResettingPassword) && (
                <div className="space-y-2">
                  <Label
                    className="text-gray-700 dark:text-gray-200"
                    htmlFor="password"
                  >
                    {isResettingPassword ? "New Password" : "Password"}
                  </Label>
                  <div className="relative">
                    <Input
                      id="password"
                      type={showPassword ? "text" : "password"}
                      placeholder={
                        isRegistering
                          ? "Create a secure password"
                          : isResettingPassword
                            ? "Enter your new password"
                            : "Enter your password"
                      }
                      value={password}
                      onChange={(e) => setPassword(e.target.value)}
                      className="bg-gray-50 dark:bg-gray-800 border-gray-300 dark:border-gray-700 text-gray-900 dark:text-white pr-10"
                    />
                    <button
                      type="button"
                      onClick={() => setShowPassword(!showPassword)}
                      className="absolute right-3 top-1/2 transform -translate-y-1/2 text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-white transition-colors"
                    >
                      {showPassword ? (
                        <EyeOff className="w-5 h-5" />
                      ) : (
                        <Eye className="w-5 h-5" />
                      )}
                    </button>
                  </div>
                </div>
              )}

              {!isRegistering && !isVerifying && !isForgotPassword && !isResettingPassword && (
                <div className="flex items-center justify-between">
                  <label className="flex items-center gap-2 text-sm cursor-pointer">
                    <input
                      type="checkbox"
                      className="rounded border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800"
                    />
                    <span className="text-gray-600 dark:text-gray-300">
                      Remember me
                    </span>
                  </label>
                  <button
                    type="button"
                    className="text-sm text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 transition-colors"
                    onClick={() => setIsForgotPassword(true)}
                  >
                    Forgot password?
                  </button>
                </div>
              )}

              <Button
                type="submit"
                className="w-full bg-blue-600 hover:bg-blue-700 text-white"
                disabled={isLoading}
              >
                {isLoading ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Please wait
                  </>
                ) : isResettingPassword
                  ? "Reset Password"
                  : isForgotPassword
                    ? "Send OTP"
                    : isVerifying 
                      ? "Verify OTP" 
                      : isRegistering 
                        ? "Create Account" 
                        : "Sign In"}
              </Button>

              {!isForgotPassword && !isResettingPassword && (
                <p className="text-center text-sm text-gray-600 dark:text-gray-400 mt-4">
                  {isVerifying ? (
                    <>
                      <button
                        type="button"
                        className="text-blue-500 hover:text-blue-400 font-medium transition-colors"
                        onClick={() => setIsVerifying(false)}
                        disabled={isLoading}
                      >
                        Back to sign up
                      </button>
                    </>
                  ) : isRegistering ? (
                    <>
                      Already have an account?{" "}
                      <button
                        type="button"
                        className="text-blue-500 hover:text-blue-400 font-medium transition-colors"
                        onClick={toggleMode}
                        disabled={isLoading}
                      >
                        Sign in
                      </button>
                    </>
                  ) : (
                    <>
                      Don't have an account?{" "}
                      <button
                        type="button"
                        className="text-blue-500 hover:text-blue-400 font-medium transition-colors"
                        onClick={toggleMode}
                        disabled={isLoading}
                      >
                        Sign up
                      </button>
                    </>
                  )}
                </p>
              )}
            </form>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
