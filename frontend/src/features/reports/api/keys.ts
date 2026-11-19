// The query keys of the feature: one report per kind and filters.
export const reportsKeys = {
  all: ['reports'] as const,
  report: (kind: string, query: object) => ['reports', kind, query] as const,
};
