-- +migrate Up

-- WITHOUT ROWID stores the row in the primary key b-tree, so authenticating is a single lookup
CREATE TABLE auth_keys_new (
    key_hash TEXT NOT NULL,
    hash_alg TEXT NOT NULL,
    user TEXT NOT NULL REFERENCES users(name) ON UPDATE CASCADE ON DELETE CASCADE,
    created_at DATETIME NOT NULL DEFAULT (unixepoch()),
    last_used DATETIME NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY (key_hash, hash_alg)
) WITHOUT ROWID;

INSERT INTO auth_keys_new (key_hash, hash_alg, user, created_at, last_used)
SELECT key_hash, 'pbkdf2-sha256_10000', user, created_at, last_used FROM auth_keys;

DROP TABLE auth_keys;
ALTER TABLE auth_keys_new RENAME TO auth_keys;

CREATE INDEX auth_keys_user_index ON auth_keys (user);

-- +migrate Down

CREATE TABLE auth_keys_old (
    key_hash TEXT PRIMARY KEY,
    user TEXT NOT NULL REFERENCES users(name) ON UPDATE CASCADE ON DELETE CASCADE,
    created_at DATETIME NOT NULL DEFAULT (unixepoch()),
    last_used DATETIME NOT NULL DEFAULT (unixepoch())
);

INSERT INTO auth_keys_old (key_hash, user, created_at, last_used)
SELECT key_hash, user, created_at, last_used FROM auth_keys WHERE hash_alg = 'pbkdf2-sha256_10000';

DROP TABLE auth_keys;
ALTER TABLE auth_keys_old RENAME TO auth_keys;
