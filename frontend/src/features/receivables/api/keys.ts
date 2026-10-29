import type { PageQuery } from './types';

// Everything about a customer's credit shares a prefix, so a payment
// refreshes its credit and its statement at once; the aging lists all.
export const receivablesKeys = {
  customer: (customerId: string) => ['receivables', 'customer', customerId] as const,
  credit: (customerId: string) => ['receivables', 'customer', customerId, 'credit'] as const,
  movements: (customerId: string, query: PageQuery) =>
    ['receivables', 'customer', customerId, 'movements', query] as const,
  agings: ['receivables', 'aging'] as const,
  aging: (query: PageQuery) => ['receivables', 'aging', query] as const,
};
