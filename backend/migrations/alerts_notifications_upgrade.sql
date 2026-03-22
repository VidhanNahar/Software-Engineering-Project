-- Adds alert rules and notifications.
-- Safe to run multiple times.

CREATE TABLE IF NOT EXISTS alerts (
    alert_id       UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID         NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    stock_id       UUID         NOT NULL REFERENCES stock(stock_id) ON DELETE CASCADE,
    target_price   NUMERIC(15,2) NOT NULL CHECK (target_price > 0),
    direction      VARCHAR(8)   NOT NULL CHECK (direction IN ('above', 'below')),
    is_active      BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    triggered_at   TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_alerts_user_id ON alerts(user_id);
CREATE INDEX IF NOT EXISTS idx_alerts_stock_id ON alerts(stock_id);
CREATE INDEX IF NOT EXISTS idx_alerts_active ON alerts(is_active);

CREATE TABLE IF NOT EXISTS notifications (
    notification_id UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID         NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    type            VARCHAR(32)  NOT NULL,
    title           VARCHAR(128) NOT NULL,
    message         TEXT         NOT NULL,
    is_read         BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notifications_user_created ON notifications(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_user_unread ON notifications(user_id, is_read);
