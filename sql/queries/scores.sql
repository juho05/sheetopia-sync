-- name: FindScore :one
SELECT * FROM scores WHERE user = ? AND id = ?;

-- name: FindAllScoreIDs :many
SELECT user, id FROM scores;

-- name: FindScoresChangedAfterWithTagIds :many
SELECT scores.*, score_tags.tag_id FROM scores
LEFT JOIN score_tags ON scores.user = score_tags.user AND scores.id = score_tags.score_id
WHERE scores.user = ? AND changed >= ? AND file_type != 'none' ORDER BY scores.id;

-- name: UpsertScore :exec
INSERT INTO scores (id, user, metadata_updated_at, file_updated_at, file_type, title, metadata_json, type, changed)
VALUES (?,?,?,0,'none',?,?,?,unixepoch())
ON CONFLICT (user, id) DO UPDATE SET metadata_updated_at = excluded.metadata_updated_at, title = excluded.title, metadata_json = excluded.metadata_json, type = excluded.type, changed = excluded.changed;

-- name: DeleteScore :execresult
DELETE FROM scores WHERE user = ? AND id = ?;

-- name: CreateDeletedScoreMarker :exec
INSERT INTO deleted_scores (score_id, user, deleted_at) VALUES (?, ?, unixepoch());

-- name: FindDeletedScoreMarker :one
SELECT * FROM deleted_scores WHERE user = ? AND score_id = ?;

-- name: FindDeletedScoresSince :many
SELECT score_id, deleted_at FROM deleted_scores WHERE user = ? AND deleted_at >= ?;

-- name: DeleteDeletedScoreMarker :exec
DELETE FROM deleted_scores WHERE user = ? AND score_id = ?;

-- name: UnassignAllTags :exec
DELETE FROM score_tags WHERE user = ? AND score_id = ?;

-- name: AssignTag :exec
INSERT INTO score_tags (user, score_id, tag_id) VALUES (?, ?, ?) ON CONFLICT (user,score_id,tag_id) DO NOTHING;

-- name: FindTag :one
SELECT * FROM tags WHERE user = ? AND id = ?;

-- name: FindTagsChangedAfter :many
SELECT * FROM tags WHERE user = ? AND changed >= ?;

-- name: UpsertTag :exec
INSERT INTO tags (id, user, updated_at, name, color, type, changed) VALUES (?, ?, ?, ?, ?, ?, unixepoch())
ON CONFLICT (user, id) DO UPDATE SET updated_at = excluded.updated_at, name = excluded.name, color = excluded.color, type = excluded.type, changed = excluded.changed;

-- name: DeleteTag :execresult
DELETE FROM tags WHERE user = ? AND id = ?;

-- name: FindDeletedTagMarker :one
SELECT * FROM deleted_tags WHERE user = ? AND tag_id = ?;

-- name: FindDeletedTagsSince :many
SELECT tag_id, deleted_at FROM deleted_tags WHERE user = ? AND deleted_at >= ?;

-- name: CreateDeletedTagMarker :exec
INSERT INTO deleted_tags (tag_id, user, deleted_at) VALUES (?, ?, unixepoch());

-- name: DeleteDeletedTagMarker :exec
DELETE FROM deleted_tags WHERE user = ? AND tag_id = ?;

-- name: GetAssignedTagIDs :many
SELECT tag_id FROM score_tags WHERE user = ? AND score_id = ?;

-- name: UpdateFileInfo :execresult
UPDATE scores SET file_updated_at = ?, file_type = ?, changed = unixepoch() WHERE user = ? AND id = ? AND file_updated_at < ?;
