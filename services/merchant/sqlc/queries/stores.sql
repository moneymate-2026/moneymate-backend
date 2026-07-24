-- name: CreateStore :one
INSERT INTO stores (
    owner_id, owner_name, contact_email, mobile_number, 
    legal_name, dba_name, business_type, tax_id, registered_address, display_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING id, display_id, status, plan, created_at;

-- name: SubmitKYC :exec
INSERT INTO kyc_documents (
    store_id, aadhaar_number, aadhaar_doc_url, shop_license_url
) VALUES (
    $1, $2, $3, $4
);

-- name: GetStoreByOwnerID :one
SELECT 
    id, display_id, legal_name, status, plan 
FROM stores 
WHERE owner_id = $1 LIMIT 1;

-- name: UpdateStoreStatus :exec
UPDATE stores 
SET status = $2, updated_at = NOW() 
WHERE id = $1;

-- name: GetPendingStores :many
SELECT 
    id, owner_name, contact_email, mobile_number, legal_name, business_type, status, created_at
FROM stores
WHERE status = 'pending_kyc'
ORDER BY created_at ASC;