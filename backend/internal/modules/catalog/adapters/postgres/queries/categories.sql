-- name: CreateCategory :exec
INSERT INTO categories (id, tenant_id, name, parent_id, created_at)
VALUES ($1, $2, $3, $4, $5);

-- name: GetCategory :one
SELECT id, tenant_id, name, parent_id, created_at
FROM categories
WHERE id = $1;

-- name: ListCategories :many
SELECT id, tenant_id, name, parent_id, created_at
FROM categories
ORDER BY name, id;

-- name: UpdateCategory :execrows
UPDATE categories
SET name = $2, parent_id = $3
WHERE id = $1;

-- name: DeleteCategory :execrows
DELETE FROM categories
WHERE id = $1;

-- CategoryAncestors returns the category and its ancestors, nearest first.
-- The depth bound stops the walk if a concurrent move ever created a cycle.
-- name: CategoryAncestors :many
WITH RECURSIVE chain (id, parent_id, depth) AS (
    SELECT c.id, c.parent_id, 1
    FROM categories c
    WHERE c.id = $1
    UNION ALL
    SELECT p.id, p.parent_id, chain.depth + 1
    FROM categories p
    JOIN chain ON p.id = chain.parent_id
    WHERE chain.depth < 100
)
SELECT chain.id::uuid
FROM chain
ORDER BY depth;
