import { useState, useEffect } from "react";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { toast } from "sonner";
import { userApi } from "../api";
import {
  User,
  ShieldCheck,
  ShieldAlert,
  Loader2,
  Mail,
  Calendar,
  Hash,
  CheckCircle2,
} from "lucide-react";

interface UserProfile {
  name: string;
  email_id: string;
  role: string;
  date_of_birth: string;
  phone_number?: string;
  is_verified_email: boolean;
  is_kyc_verified: boolean;
  aadhar_id?: string;
  pan_id?: string;
}

export default function Profile() {
  const [profile, setProfile] = useState<UserProfile | null>(null);
  const [loading, setLoading] = useState(true);
  const [kycLoading, setKycLoading] = useState(false);
  const [formData, setFormData] = useState({
    aadhar_id: "",
    pan_id: "",
  });

  const fetchProfile = async () => {
    try {
      const userId = localStorage.getItem("user_id");
      if (!userId) {
        toast.error("User not found in session");
        setLoading(false);
        return;
      }
      const data = await userApi.getProfile(userId);
      setProfile(data);
      if (data) {
        setFormData({
          aadhar_id: data.aadhar_id || "",
          pan_id: data.pan_id || "",
        });
      }
    } catch (error: unknown) {
      toast.error(
        error instanceof Error ? error.message : "Failed to load profile",
      );
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchProfile();
  }, []);

  const handleKycSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!formData.aadhar_id && !formData.pan_id) {
      toast.error("Please provide at least one KYC document (Aadhar or PAN)");
      return;
    }

    try {
      setKycLoading(true);
      const res = await userApi.completeKyc({
        aadhar_id: formData.aadhar_id || undefined,
        pan_id: formData.pan_id || undefined,
      });
      toast.success(res.message || "KYC completed successfully!");
      // Update local storage role if needed, since KYC promotes guest -> user
      if (res.user?.role) {
        localStorage.setItem("user_role", res.user.role);
      }
      await fetchProfile();
    } catch (error: unknown) {
      toast.error(
        error instanceof Error ? error.message : "Failed to submit KYC",
      );
    } finally {
      setKycLoading(false);
    }
  };

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    setFormData((prev) => ({ ...prev, [name]: value }));
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full">
        <Loader2 className="w-8 h-8 animate-spin text-blue-500" />
      </div>
    );
  }

  if (!profile) {
    return (
      <div className="flex items-center justify-center h-full text-muted-foreground">
        Failed to load profile information.
      </div>
    );
  }

  return (
    <div className="space-y-6 max-w-5xl mx-auto">
      <div>
        <h1 className="text-3xl font-bold text-foreground">My Profile</h1>
        <p className="text-muted-foreground mt-1">
          Manage your account details and verification status
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* User Details Card */}
        <Card className="border border-border">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <User className="w-5 h-5 text-blue-500" />
              Account Details
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-6">
            <div className="flex items-center gap-4 p-4 bg-muted/30 rounded-lg">
              <div className="w-16 h-16 rounded-full bg-blue-600 flex items-center justify-center text-white text-2xl font-bold uppercase">
                {profile.name?.substring(0, 2) || "U"}
              </div>
              <div>
                <h3 className="text-xl font-semibold text-foreground">
                  {profile.name}
                </h3>
                <div className="flex items-center gap-2 mt-1">
                  <span className="px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800 dark:bg-blue-900/50 dark:text-blue-300 capitalize">
                    {profile.role}
                  </span>
                  {profile.is_verified_email && (
                    <span className="flex items-center text-xs text-green-600 dark:text-green-400 font-medium">
                      <CheckCircle2 className="w-3 h-3 mr-1" />
                      Email Verified
                    </span>
                  )}
                </div>
              </div>
            </div>

            <div className="space-y-4">
              <div className="flex items-center gap-3 text-sm">
                <Mail className="w-4 h-4 text-muted-foreground" />
                <div>
                  <p className="text-muted-foreground text-xs">Email Address</p>
                  <p className="font-medium text-foreground">
                    {profile.email_id}
                  </p>
                </div>
              </div>

              <div className="flex items-center gap-3 text-sm">
                <Calendar className="w-4 h-4 text-muted-foreground" />
                <div>
                  <p className="text-muted-foreground text-xs">Date of Birth</p>
                  <p className="font-medium text-foreground">
                    {new Date(profile.date_of_birth).toLocaleDateString()}
                  </p>
                </div>
              </div>

              {profile.phone_number && (
                <div className="flex items-center gap-3 text-sm">
                  <Hash className="w-4 h-4 text-muted-foreground" />
                  <div>
                    <p className="text-muted-foreground text-xs">
                      Phone Number
                    </p>
                    <p className="font-medium text-foreground">
                      {profile.phone_number}
                    </p>
                  </div>
                </div>
              )}
            </div>
          </CardContent>
        </Card>

        {/* KYC Verification Card */}
        <Card className="border border-border">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              {profile.is_kyc_verified ? (
                <ShieldCheck className="w-5 h-5 text-green-500" />
              ) : (
                <ShieldAlert className="w-5 h-5 text-amber-500" />
              )}
              Identity Verification (KYC)
            </CardTitle>
            <CardDescription>
              {profile.is_kyc_verified
                ? "Your identity has been successfully verified."
                : "Complete your KYC to unlock real trading capabilities."}
            </CardDescription>
          </CardHeader>
          <CardContent>
            {profile.is_kyc_verified ? (
              <div className="flex flex-col items-center justify-center p-6 bg-green-50 dark:bg-green-900/10 rounded-lg border border-green-200 dark:border-green-900/50 text-center">
                <div className="w-16 h-16 bg-green-100 dark:bg-green-900/30 rounded-full flex items-center justify-center mb-4">
                  <CheckCircle2 className="w-8 h-8 text-green-600 dark:text-green-400" />
                </div>
                <h3 className="text-lg font-semibold text-green-800 dark:text-green-300">
                  KYC Verified
                </h3>
                <p className="text-sm text-green-600/80 dark:text-green-400/80 mt-1 max-w-[250px]">
                  Your account is fully verified and ready for trading.
                </p>
                {(profile.aadhar_id || profile.pan_id) && (
                  <div className="mt-6 w-full space-y-2 text-left bg-background/50 p-3 rounded text-sm">
                    {profile.aadhar_id && (
                      <div className="flex justify-between">
                        <span className="text-muted-foreground">
                          Aadhar ID:
                        </span>
                        <span className="font-mono text-foreground">
                          •••• {profile.aadhar_id.slice(-4)}
                        </span>
                      </div>
                    )}
                    {profile.pan_id && (
                      <div className="flex justify-between">
                        <span className="text-muted-foreground">PAN ID:</span>
                        <span className="font-mono text-foreground">
                          •••••• {profile.pan_id.slice(-4)}
                        </span>
                      </div>
                    )}
                  </div>
                )}
              </div>
            ) : (
              <form onSubmit={handleKycSubmit} className="space-y-4">
                <div className="bg-amber-50 dark:bg-amber-950/30 text-amber-800 dark:text-amber-200 text-sm p-3 rounded border border-amber-200 dark:border-amber-900/50 mb-4">
                  <strong>Action Required:</strong> Provide your Aadhar or PAN
                  to upgrade your account from Guest to Trader.
                </div>

                <div className="space-y-2">
                  <Label htmlFor="aadhar_id">Aadhar Number</Label>
                  <Input
                    id="aadhar_id"
                    name="aadhar_id"
                    placeholder="Enter 12-digit Aadhar number"
                    value={formData.aadhar_id}
                    onChange={handleInputChange}
                    className="font-mono"
                  />
                </div>

                <div className="space-y-2">
                  <Label htmlFor="pan_id">PAN Number</Label>
                  <Input
                    id="pan_id"
                    name="pan_id"
                    placeholder="Enter 10-character PAN"
                    value={formData.pan_id}
                    onChange={handleInputChange}
                    className="font-mono uppercase"
                  />
                </div>

                <Button
                  type="submit"
                  className="w-full bg-blue-600 hover:bg-blue-700 text-white mt-2"
                  disabled={
                    kycLoading || (!formData.aadhar_id && !formData.pan_id)
                  }
                >
                  {kycLoading ? (
                    <>
                      <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                      Submitting...
                    </>
                  ) : (
                    "Submit KYC Documents"
                  )}
                </Button>
              </form>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
