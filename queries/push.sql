-- name: upsert-push-token
INSERT INTO push_tokens (uuid, token, label, is_active, updated_at)
VALUES ($1, $2, $3, true, NOW())
ON CONFLICT (token) DO UPDATE
	SET label = EXCLUDED.label, is_active = true, updated_at = NOW()
RETURNING id;

-- name: get-push-tokens
SELECT id, uuid, token, label, is_active, created_at, updated_at
FROM push_tokens ORDER BY created_at DESC;

-- name: get-active-push-tokens
SELECT token FROM push_tokens WHERE is_active = true ORDER BY created_at DESC;

-- name: deactivate-push-token
UPDATE push_tokens SET is_active = false, updated_at = NOW() WHERE token = $1;

-- name: delete-push-token
DELETE FROM push_tokens WHERE id = $1;
