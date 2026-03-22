-- Upgrade users table for role-based access and KYC gating.
-- Safe to run multiple times.

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_role') THEN
        CREATE TYPE user_role AS ENUM ('guest', 'user', 'admin');
    END IF;
END$$;

ALTER TABLE users ADD COLUMN IF NOT EXISTS role user_role NOT NULL DEFAULT 'guest';
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_kyc_verified BOOLEAN NOT NULL DEFAULT FALSE;

-- Existing verified users become regular users by default.
UPDATE users
SET role = 'user', is_kyc_verified = TRUE
WHERE is_verified_email = TRUE
  AND (role = 'guest' OR role IS NULL);
