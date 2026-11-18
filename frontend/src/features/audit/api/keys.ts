import type { AuditQuery } from './auditEvents';

// The query keys of the feature: one page of events per filter.
export const auditKeys = {
  all: ['audit'] as const,
  events: (query: AuditQuery) => ['audit', 'events', query] as const,
};
