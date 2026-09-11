-- name: CreatePod :one
INSERT INTO payment.pods (id, account_id, user_id, name, target_amount, target_date, icon, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, account_id, user_id, name, target_amount, target_date, icon, status, created_at, updated_at;

-- name: GetPodByID :one
SELECT id, account_id, user_id, name, target_amount, target_date, icon, status, created_at, updated_at
FROM payment.pods
WHERE id = $1;

-- name: ListPodsByUser :many
SELECT id, account_id, user_id, name, target_amount, target_date, icon, status, created_at, updated_at
FROM payment.pods
WHERE user_id = $1
ORDER BY created_at;

-- name: UpdatePod :one
UPDATE payment.pods
SET name = $2,
    target_amount = $3,
    target_date = $4,
    icon = $5,
    status = $6,
    updated_at = NOW()
WHERE id = $1
RETURNING id, account_id, user_id, name, target_amount, target_date, icon, status, created_at, updated_at;

-- name: DeletePod :exec
DELETE FROM payment.pods
WHERE id = $1;
