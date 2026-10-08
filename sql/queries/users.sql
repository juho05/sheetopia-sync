-- name: CreateUser :exec
INSERT INTO users (name, password_hash) VALUES (?, ?);

-- name: FindUser :one
SELECT * FROM users WHERE name = ?;

-- name: FindUsers :many
SELECT * FROM users;

-- name: DeleteUser :execresult
DELETE FROM users WHERE name = ?;

-- name: UpdateUserName :execresult
UPDATE users SET name = ? WHERE name = ?;

-- name: UpdateUserPassword :execresult
UPDATE users SET password_hash = ? WHERE name = ?;

-- name: CreateAuthKey :exec
INSERT INTO auth_keys (key_hash, hash_alg, user) VALUES (?, ?, ?);

-- name: VerifyAuthKey :one
UPDATE auth_keys SET last_used = unixepoch() WHERE key_hash = ? AND hash_alg = ? RETURNING user;

-- name: MigrateAuthKey :exec
UPDATE auth_keys SET key_hash = sqlc.arg(new_key_hash), hash_alg = sqlc.arg(new_hash_alg)
WHERE key_hash = sqlc.arg(old_key_hash) AND hash_alg = sqlc.arg(old_hash_alg);

-- name: DeleteAuthKey :exec
DELETE FROM auth_keys WHERE auth_keys.key_hash = ? AND hash_alg = ?;