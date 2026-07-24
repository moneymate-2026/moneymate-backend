-- Default admin user (email: admin@moneymate.com, password: admin@5678)
INSERT INTO auth.users (
    id, email, full_name, handle, password_hash,
    status, token_version, is_email_verified, is_phone_verified,
    created_at, updated_at
) VALUES (
    'b0000000-0000-0000-0000-000000000001',
    'admin@moneymate.com',
    'Platform Admin',
    'admin',
    '$argon2id$v=19$m=65536,t=1,p=16$oEn99k/DlFoIG0G3LeCvgA$ddAGQEnPhaGovhsF3pMWTDinCFCe0Tw7yYj3xci8sVE',
    'active',
    0,
    true,
    false,
    NOW(),
    NOW()
);

-- Assign admin role
INSERT INTO auth.user_roles (user_id, role_id, assigned_at)
VALUES (
    'b0000000-0000-0000-0000-000000000001',
    'a0000000-0000-0000-0000-000000000003',
    NOW()
);