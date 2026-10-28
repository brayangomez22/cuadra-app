import type { components } from '@/shared/api/schema';

export type Location = components['schemas']['Location'];
export type StockLevel = components['schemas']['StockLevel'];
export type ReorderAlert = components['schemas']['ReorderAlert'];
export type StockMovement = components['schemas']['StockMovement'];
export type StockCount = components['schemas']['StockCount'];
export type StockCountLine = components['schemas']['StockCountLine'];
export type StockCountLineStatus = components['schemas']['StockCountLineStatus'];
export type StockCountSummary = components['schemas']['StockCountSummary'];
export type CountEntry = components['schemas']['CountEntry'];

/** Rows per page of the stock and the kardex. */
export const PAGE_SIZE = 20;
