CREATE TABLE payment.pods (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id    UUID NOT NULL UNIQUE REFERENCES payment.accounts(id),
    user_id       UUID NOT NULL,
    name          TEXT NOT NULL,
    target_amount BIGINT,
    target_date   DATE,
    icon          TEXT,
    status        TEXT NOT NULL DEFAULT 'active',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_pods_user_id ON payment.pods(user_id);
