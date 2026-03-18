CREATE EXTENSION IF NOT EXISTS "pgcrypto"; -- needed for gen_random_uuid()

CREATE TABLE IF NOT EXISTS users (
    user_id        UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    name           VARCHAR(255)  NOT NULL,
    email_id       VARCHAR(255)  UNIQUE,
    password       VARCHAR(255)  NOT NULL,
    aadhar_id      CHAR(12)      UNIQUE,         -- fixed 12 digits
    pan_id         CHAR(10)      UNIQUE,         -- fixed 10 chars
    phone_number   CHAR(10)      UNIQUE,         -- fixed 10 digits
    date_of_birth  DATE          NOT NULL,
    is_verified_email BOOLEAN    NOT NULL DEFAULT FALSE
);
