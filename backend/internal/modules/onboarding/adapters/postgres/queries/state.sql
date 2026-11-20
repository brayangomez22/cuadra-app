-- name: TenantState :one
-- What the getting-started checklist reads of a tenant: its business
-- template and, for each step, whether its data exists. Each fact is an
-- EXISTS, so it stops at the first row.
SELECT
    t.business_template,
    (t.address <> '' AND t.city <> '' AND t.phone <> '')::boolean AS has_contact,
    EXISTS (SELECT 1 FROM products p WHERE p.tenant_id = t.id) AS has_product,
    EXISTS (SELECT 1 FROM stock_movements m WHERE m.tenant_id = t.id) AS has_stock,
    EXISTS (SELECT 1 FROM variant_units u WHERE u.tenant_id = t.id) AS has_alternate_unit,
    EXISTS (
        SELECT 1 FROM product_variants v
        WHERE v.tenant_id = t.id
        GROUP BY v.product_id
        HAVING count(*) > 1
    ) AS has_variant_product,
    EXISTS (SELECT 1 FROM reorder_rules r WHERE r.tenant_id = t.id) AS has_reorder_rule,
    EXISTS (SELECT 1 FROM credit_accounts c WHERE c.tenant_id = t.id AND c.credit_limit > 0) AS has_credit_customer,
    (
        (SELECT count(*) FROM users u WHERE u.tenant_id = t.id AND u.active) > 1
        OR EXISTS (
            SELECT 1 FROM user_invitations i
            WHERE i.tenant_id = t.id AND i.accepted_at IS NULL AND i.revoked_at IS NULL
              AND i.expires_at > sqlc.arg(now)::timestamptz
        )
    )::boolean AS has_teammate,
    EXISTS (SELECT 1 FROM sales s WHERE s.tenant_id = t.id) AS has_sale
FROM tenants t
WHERE t.id = sqlc.arg(tenant_id);
