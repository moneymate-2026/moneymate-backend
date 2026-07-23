-- name: CreateUser :one
INSERT INTO auth.users (
    id,
    email,
    phone,
    full_name,
    handle,
    password_hash
)
VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;



-- name: GetUserByID :one
SELECT * FROM auth.users
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM auth.users
WHERE email = $1;

-- name: GetUserByPhone :one
SELECT * FROM auth.users
WHERE phone = $1;

-- name: GetUserByHandle :one
SELECT * FROM auth.users
WHERE handle = $1;

-- name: VerifyEmail :exec
UPDATE auth.users
SET
    is_email_verified  = TRUE,
    status = 'active',
    updated_at = NOW()
WHERE id = $1;

-- name: VerifyPhone :exec
UPDATE auth.users
SET
    is_phone_verified = TRUE,
    updated_at = NOW()
WHERE id = $1;

-- name: UpdatePassword :exec
UPDATE auth.users
SET
    password_hash = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: UpdateUserStatus :exec 
UPDATE auth.users
SET
    status = $2,
    updated_at = NOW()
WHERE id = $1;


-- name: IncrementTokenVersion :one
UPDATE auth.users
SET
    token_version = token_version + 1,
    updated_at = NOW()
WHERE id = $1
RETURNING token_version;

-- name: GetTokenVersion :one
SELECT token_version FROM auth.users
WHERE id = $1;

-- name: SoftDeleteUser :exec
UPDATE auth.users
SET
    status = 'deleted',
    updated_at = NOW()
WHERE id = $1;

-- name: HandleExists :one
SELECT EXISTS(
    SELECT 1 FROM auth.users WHERE handle = $1
) AS exists;

-- name: EmailExists :one
SELECT EXISTS(
    SELECT 1 FROM auth.users WHERE email = $1
) AS exists;

-- name: PhoneExists :one
SELECT EXISTS(
    SELECT 1 FROM auth.users WHERE phone = $1
) AS exists;


-- name: ListUsers :many
SELECT * FROM auth.users
WHERE
    (sqlc.narg('status')::auth_user_status IS NULL OR status = sqlc.narg('status'))
    AND (sqlc.narg('role')::text IS NULL OR role = sqlc.narg('role'))
    AND (
        sqlc.narg('search')::text IS NULL
        OR email ILIKE '%' || sqlc.narg('search')::text || '%'
        OR full_name ILIKE '%' || sqlc.narg('search')::text || '%'
        OR handle ILIKE '%' || sqlc.narg('search')::text || '%'
    )
ORDER BY
    CASE WHEN sqlc.arg('sort_by')::text = 'email' AND NOT sqlc.arg('sort_desc')::bool THEN email END ASC NULLS LAST,
    CASE WHEN sqlc.arg('sort_by')::text = 'email' AND sqlc.arg('sort_desc')::bool THEN email END DESC NULLS LAST,
    CASE WHEN sqlc.arg('sort_by')::text = 'full_name' AND NOT sqlc.arg('sort_desc')::bool THEN full_name END ASC NULLS LAST,
    CASE WHEN sqlc.arg('sort_by')::text = 'full_name' AND sqlc.arg('sort_desc')::bool THEN full_name END DESC NULLS LAST,
    CASE WHEN sqlc.arg('sort_by')::text = 'created_at' AND NOT sqlc.arg('sort_desc')::bool THEN created_at END ASC,
    CASE WHEN sqlc.arg('sort_by')::text = 'created_at' AND sqlc.arg('sort_desc')::bool THEN created_at END DESC,
    created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountUsers :one
SELECT COUNT(*) FROM auth.users
WHERE
    (sqlc.narg('status')::auth_user_status IS NULL OR status = sqlc.narg('status'))
    AND (sqlc.narg('role')::text IS NULL OR role = sqlc.narg('role'))
    AND (
        sqlc.narg('search')::text IS NULL
        OR email ILIKE '%' || sqlc.narg('search')::text || '%'
        OR full_name ILIKE '%' || sqlc.narg('search')::text || '%'
        OR handle ILIKE '%' || sqlc.narg('search')::text || '%'
    );