-- name: CreateFeedback :exec
INSERT INTO feedback (id, tenant_id, user_id, message, screen, app_version, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: GetFeedback :one
SELECT id, tenant_id, user_id, message, screen, app_version, created_at, notified_at
FROM feedback
WHERE tenant_id = $1 AND id = $2;

-- name: MarkFeedbackNotified :exec
UPDATE feedback SET notified_at = $3 WHERE tenant_id = $1 AND id = $2;
