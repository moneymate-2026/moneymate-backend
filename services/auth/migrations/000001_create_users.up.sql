CREATE TYPE auth.user_status AS ENUM (
    'pending',
    'active',
    'suspended',
    'deleted'
);

CREATE TABLE auth.users (
    id UUID PRIMARY KEY,

    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash TEXT,

    email_verified BOOLEAN NOT NULL DEFAULT FALSE,

    status auth.user_status NOT NULL DEFAULT 'pending',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);