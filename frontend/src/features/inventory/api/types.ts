import type { components } from '@/shared/api/schema';

export type Location = components['schemas']['Location'];
export type StockLevel = components['schemas']['StockLevel'];
export type ReorderAlert = components['schemas']['ReorderAlert'];
export type StockMovement = components['schemas']['StockMovement'];

/** Rows per page of the stock and the kardex. */
export const PAGE_SIZE = 20;
