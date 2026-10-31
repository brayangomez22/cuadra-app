export const salesKeys = {
  sales: ['sales', 'list'] as const,
  saleList: (query: unknown) => ['sales', 'list', query] as const,
  sale: (id: string) => ['sales', 'detail', id] as const,
  cashSessions: ['sales', 'cash-sessions'] as const,
  discountLimit: ['sales', 'discount-limit'] as const,
  currentCash: ['sales', 'cash-sessions', 'current'] as const,
  cashSession: (id: string) => ['sales', 'cash-sessions', id] as const,
};
