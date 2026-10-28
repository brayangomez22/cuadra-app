import type { CustomerQuery } from './types';

// The customer lists share a prefix so a saved customer invalidates every page.
export const customersKeys = {
  lists: ['customers', 'list'] as const,
  list: (query: CustomerQuery) => ['customers', 'list', query] as const,
  customer: (id: string) => ['customers', 'customer', id] as const,
  segments: ['customers', 'segments'] as const,
};
