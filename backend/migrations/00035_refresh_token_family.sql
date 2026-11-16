-- Session lifetime (T38, P06): each refresh token keeps when its family (the
-- session one login started) began, so a rotation never extends a session
-- past its maximum age. Existing tokens take the creation of the oldest
-- token of their family. Also indexes the expiry, for the nightly purge of
-- expired tokens (P07).

-- +goose Up
ALTER TABLE refresh_tokens ADD COLUMN family_started_at timestamptz;

-- FORCE ROW LEVEL SECURITY applies to the owner too, and the owner running
-- migrations on Azure is not a superuser: with no tenant set, the backfill
-- would see zero rows. FORCE is lifted for the backfill and its check, and
-- restored in the same transaction.
ALTER TABLE refresh_tokens NO FORCE ROW LEVEL SECURITY;

UPDATE refresh_tokens r
SET family_started_at = f.started_at
FROM (SELECT tenant_id, family_id, min(created_at) AS started_at
      FROM refresh_tokens GROUP BY tenant_id, family_id) f
WHERE r.tenant_id = f.tenant_id AND r.family_id = f.family_id;

-- +goose StatementBegin
DO $$
DECLARE
    token_count  bigint;
    filled_count bigint;
BEGIN
    SELECT count(*) INTO token_count FROM refresh_tokens;
    SELECT count(*) INTO filled_count FROM refresh_tokens
    WHERE family_started_at IS NOT NULL AND family_started_at <= created_at;
    IF token_count <> filled_count THEN
        RAISE EXCEPTION 'refresh_tokens backfill: % tokens but % with the start of their family', token_count, filled_count;
    END IF;
END
$$;
-- +goose StatementEnd

ALTER TABLE refresh_tokens FORCE ROW LEVEL SECURITY;

ALTER TABLE refresh_tokens ALTER COLUMN family_started_at SET NOT NULL;
ALTER TABLE refresh_tokens ADD CONSTRAINT refresh_tokens_family_started_check CHECK (family_started_at <= created_at);

CREATE INDEX refresh_tokens_expires_idx ON refresh_tokens (expires_at);

-- The purge works tenant by tenant, within each one's Row-Level Security:
-- this function only tells which tenants have expired tokens. It runs as
-- auth_lookup (00003), which may read just the tenant and the expiry.
GRANT SELECT (tenant_id, expires_at) ON refresh_tokens TO auth_lookup;

CREATE POLICY expired_lookup ON refresh_tokens
    FOR SELECT TO auth_lookup
    USING (true);

CREATE FUNCTION tenants_with_expired_refresh_tokens(p_before timestamptz, p_limit int)
    RETURNS TABLE (tenant_id uuid)
    LANGUAGE sql STABLE SECURITY DEFINER
    SET search_path = pg_catalog, pg_temp
    AS $$
        SELECT DISTINCT r.tenant_id
        FROM public.refresh_tokens r
        WHERE r.expires_at < p_before
        ORDER BY r.tenant_id
        LIMIT p_limit
    $$;

-- A new owner needs CREATE on the schema, which 00001 took from PUBLIC; a
-- superuser skips that check, the migration role on Azure does not.
GRANT CREATE ON SCHEMA public TO auth_lookup;
ALTER FUNCTION tenants_with_expired_refresh_tokens(timestamptz, int) OWNER TO auth_lookup;
REVOKE CREATE ON SCHEMA public FROM auth_lookup;
REVOKE ALL ON FUNCTION tenants_with_expired_refresh_tokens(timestamptz, int) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION tenants_with_expired_refresh_tokens(timestamptz, int) TO app_user;

-- +goose Down
DROP FUNCTION tenants_with_expired_refresh_tokens(timestamptz, int);
DROP POLICY expired_lookup ON refresh_tokens;
REVOKE SELECT (tenant_id, expires_at) ON refresh_tokens FROM auth_lookup;
DROP INDEX refresh_tokens_expires_idx;
ALTER TABLE refresh_tokens DROP COLUMN family_started_at;
