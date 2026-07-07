CREATE TABLE auth.email_verifications (
    id UUID PRIMARY KEY,

    user_id UUID NOT NULL,

    token_hash TEXT NOT NULL UNIQUE,

    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    FOREIGN KEY (user_id)
        REFERENCES auth.users(id)
        ON DELETE CASCADE
);

CREATE INDEX idx_email_verifications_user_id
ON auth.email_verifications(user_id);