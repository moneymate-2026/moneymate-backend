CREATE TABLE auth.roles (
    id UUID PRIMARY KEY,

    name VARCHAR(50) NOT NULL UNIQUE,
    description TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);