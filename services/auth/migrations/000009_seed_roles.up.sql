CREATE EXTENSION IF NOT EXISTS pgcrypto;

INSERT INTO auth.roles (id, name, description) VALUES
    (gen_random_uuid(), 'user', 'Standard end user account'),
    (gen_random_uuid(), 'merchant', 'Merchant/business account'),
    (gen_random_uuid(), 'admin', 'Internal administrator account')
ON CONFLICT (name) DO NOTHING;