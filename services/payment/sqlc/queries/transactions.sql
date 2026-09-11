-- name: InsertTransaction :one
INSERT INTO payment.transactions
    (id, from_account_id, to_account_id, amount, status, idempotency_key, description, category_id, completed_at)
VALUES
    ($1, $2, $3, $4, $5::payment.tx_status, $6, $7, $8, $9)
RETURNING id, from_account_id, to_account_id, amount, status::text AS status,
          idempotency_key, COALESCE(description, '') AS description, category_id, created_at, completed_at;

-- name: GetTransactionByID :one
SELECT id, from_account_id, to_account_id, amount, status::text AS status,
       idempotency_key, COALESCE(description, '') AS description, category_id, created_at, completed_at
FROM payment.transactions
WHERE id = $1;

-- name: GetTransactionByIdempotencyKey :one
SELECT id, from_account_id, to_account_id, amount, status::text AS status,
       idempotency_key, COALESCE(description, '') AS description, category_id, created_at, completed_at
FROM payment.transactions
WHERE idempotency_key = $1 AND from_account_id = $2;

-- name: ListTransactionsByAccount :many
SELECT t.id, t.from_account_id, t.to_account_id, t.amount, t.status::text AS status,
       t.idempotency_key, COALESCE(t.description, '') AS description, t.category_id, t.created_at, t.completed_at
FROM payment.transactions t
JOIN payment.journal_entries j ON j.transaction_id = t.id
WHERE j.account_id = $1
ORDER BY t.created_at DESC;


-- name: UpdateTransactionStatus :exec
UPDATE payment.transactions
SET status = $2::payment.tx_status,
    completed_at = CASE WHEN $2 = 'completed' THEN NOW() ELSE completed_at END
WHERE id = $1;



-- name: ListTransactionsByAccountPaginated :many
SELECT * FROM payment.transactions
WHERE (from_account_id = @account_id::uuid OR to_account_id = @account_id::uuid)
  AND (sqlc.narg('category_id')::uuid IS NULL OR category_id = sqlc.narg('category_id'))
  AND (sqlc.narg('from_date')::timestamptz IS NULL OR created_at >= sqlc.narg('from_date'))
  AND (sqlc.narg('to_date')::timestamptz IS NULL OR created_at < sqlc.narg('to_date'))
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountTransactionsByAccount :one
SELECT COUNT(*) FROM payment.transactions
WHERE (from_account_id = @account_id::uuid OR to_account_id = @account_id::uuid)
  AND (sqlc.narg('category_id')::uuid IS NULL OR category_id = sqlc.narg('category_id'))
  AND (sqlc.narg('from_date')::timestamptz IS NULL OR created_at >= sqlc.narg('from_date'))
  AND (sqlc.narg('to_date')::timestamptz IS NULL OR created_at < sqlc.narg('to_date'));


-- name: GetSpendByCategory :many
SELECT 
    COALESCE(c.name, 'uncategorized')::text AS category,
    COUNT(*)::bigint AS transaction_count,
    COALESCE(SUM(t.amount), 0)::bigint AS total_amount
FROM payment.transactions t
LEFT JOIN payment.categories c ON t.category_id = c.id
WHERE t.from_account_id = @from_account_id::uuid
  AND t.status = 'completed'
  AND (sqlc.narg('created_at_from')::timestamptz IS NULL OR t.created_at >= sqlc.narg('created_at_from'))
  AND t.created_at < @created_at_to::timestamptz
GROUP BY c.name
ORDER BY total_amount DESC;

-- name: GetSpendByPeriod :many
SELECT 
    DATE_TRUNC(@granularity::text, t.created_at)::timestamptz AS period,
    COALESCE(SUM(t.amount), 0)::bigint AS total_amount,
    COUNT(*)::bigint AS transaction_count
FROM payment.transactions t
WHERE t.from_account_id = @from_account_id::uuid
  AND t.status = 'completed'
  AND (sqlc.narg('created_at_from')::timestamptz IS NULL OR t.created_at >= sqlc.narg('created_at_from'))
  AND t.created_at < @created_at_to::timestamptz
GROUP BY DATE_TRUNC(@granularity::text, t.created_at)
ORDER BY period ASC;