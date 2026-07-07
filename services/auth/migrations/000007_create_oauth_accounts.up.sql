CREATE TABLE auth.oauth_accounts (
    id UUID PRIMARY KEY,

    user_id UUID NOT NULL,

    provider VARCHAR(50) NOT NULL,
    provider_user_id VARCHAR(255) NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(provider, provider_user_id),

    FOREIGN KEY (user_id)
        REFERENCES auth.users(id)
        ON DELETE CASCADE
);

CREATE INDEX idx_oauth_accounts_user_id
ON auth.oauth_accounts(user_id);