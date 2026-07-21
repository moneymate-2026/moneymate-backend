-- 1. Create Enums based on UI selection criteria
CREATE TYPE merchant_status AS ENUM (
    'pending_kyc',
    'active',
    'suspended',
    'deleted'
);

CREATE TYPE business_type AS ENUM (
    'limited_liability_company',  -- UI: Limited Liability Company (LLC)
    'corporation',                -- UI: Corporation
    'sole_proprietorship',        -- UI: Sole Proprietorship
    'partnership',
    'other'
);

CREATE TYPE subscription_plan AS ENUM (
    'essential',                  -- UI: Essential Plan
    'growth',                     -- UI: Growth Plan
    'enterprise'                  -- UI: Enterprise Plan
);

-- 2. Core Stores Table (Step 1 & Step 2 of Registration)
CREATE TABLE stores (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id            UUID NOT NULL, -- Foreign key referencing auth.users(id) across services
    
    -- Owner Info (Step 2)
    owner_name          VARCHAR(255) NOT NULL, 
    contact_email       VARCHAR(255) NOT NULL UNIQUE, 
    mobile_number       VARCHAR(20) NOT NULL UNIQUE,
    
    -- Business Details (Step 1)
    legal_name          VARCHAR(255) NOT NULL,
    dba_name            VARCHAR(255),
    type                business_type NOT NULL,
    tax_id              VARCHAR(100),
    
    -- System Generated (QR & State)
    display_id          VARCHAR(20) NOT NULL UNIQUE, -- UI: ID: MM-9823-XA
    status              merchant_status NOT NULL DEFAULT 'pending_kyc',
    plan                subscription_plan NOT NULL DEFAULT 'essential', --
    
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 3. KYC Documents Table (Step 3 of Registration)
CREATE TABLE kyc_documents (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    
    -- KYC Details
    aadhaar_number      VARCHAR(12) NOT NULL UNIQUE, 
    aadhaar_doc_url     TEXT NOT NULL, 
    shop_license_url    TEXT NOT NULL, 
    
    is_verified         BOOLEAN NOT NULL DEFAULT FALSE,
    verified_at         TIMESTAMPTZ,
    
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 4. Campaigns / Offers Table
CREATE TABLE campaigns (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    
    -- Campaign Setup
    name                VARCHAR(255) NOT NULL,
    offer_type          VARCHAR(100) NOT NULL, 
    reward_value        NUMERIC(10, 2) NOT NULL,
    min_bill_amount     NUMERIC(10, 2) NOT NULL DEFAULT 0.00,
    target_audience     VARCHAR(100) NOT NULL,
    
    start_date          DATE NOT NULL,
    end_date            DATE NOT NULL,
    is_active           BOOLEAN NOT NULL DEFAULT TRUE,
    
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 5. Performance Indices
CREATE INDEX idx_stores_owner_id ON stores(owner_id);
CREATE INDEX idx_stores_display_id ON stores(display_id);
CREATE INDEX idx_stores_contact_email ON stores(contact_email);
CREATE INDEX idx_kyc_store_id ON kyc_documents(store_id);
CREATE INDEX idx_campaigns_store_id_active ON campaigns(store_id) WHERE is_active = TRUE;