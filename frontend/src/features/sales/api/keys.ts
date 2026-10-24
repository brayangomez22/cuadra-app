export const salesKeys = {
  discountLimit: ['sales', 'discount-limit'] as const,
  currentCash: ['sales', 'cash-sessions', 'current'] as const,
  cashSession: (id: string) => ['sales', 'cash-sessions', id] as const,
};
