DELETE FROM auth.roles WHERE name IN ('user', 'merchant', 'admin');
DROP EXTENSION IF EXISTS pgcrypto;