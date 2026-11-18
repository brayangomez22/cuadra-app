import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { unwrap } from '@/shared/api/errors';
import type { operations } from '@/shared/api/schema';
import { auditKeys } from './keys';

/** The filters and page of GET /audit-events. */
export type AuditQuery = NonNullable<operations['listAuditEvents']['parameters']['query']>;

export const AUDIT_PAGE_SIZE = 50;

/** One page of the audit log, newest first (only the owner: audit:read). */
export function useAuditEvents(query: AuditQuery) {
  return useQuery({
    queryKey: auditKeys.events(query),
    queryFn: async () => unwrap(await api.GET('/api/v1/audit-events', { params: { query } })),
    placeholderData: keepPreviousData,
  });
}
