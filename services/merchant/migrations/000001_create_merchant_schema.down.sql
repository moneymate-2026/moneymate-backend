-- 1. Drop Indexes (Optional but good practice for clean state)
DROP INDEX IF EXISTS idx_campaigns_store_id_active;
DROP INDEX IF EXISTS idx_kyc_store_id;
DROP INDEX IF EXISTS idx_stores_contact_email;
DROP INDEX IF EXISTS idx_stores_display_id;
DROP INDEX IF EXISTS idx_stores_owner_id;

-- 2. Drop Tables (Must drop dependent tables first to avoid FK constraint errors)
DROP TABLE IF EXISTS campaigns;
DROP TABLE IF EXISTS kyc_documents;
DROP TABLE IF EXISTS stores;

-- 3. Drop ENUM Types (Must drop after tables that reference them)
DROP TYPE IF EXISTS subscription_plan;
DROP TYPE IF EXISTS merchant_status;