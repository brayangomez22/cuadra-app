import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { unwrap } from '@/shared/api/errors';
import type { operations } from '@/shared/api/schema';
import { reportsKeys } from './keys';

type QueryOf<O extends keyof operations> = NonNullable<operations[O]['parameters']['query']>;

export type SalesQuery = QueryOf<'getSalesReport'>;
export type ProductsQuery = QueryOf<'getProductsReport'>;
export type SaleProfitsQuery = QueryOf<'getSaleProfitsReport'>;
export type IdleStockQuery = QueryOf<'getIdleStockReport'>;
export type CashClosingsQuery = QueryOf<'getCashClosingsReport'>;
export type TeamControlQuery = QueryOf<'getTeamControlReport'>;
export type StockAdjustmentsQuery = QueryOf<'getStockAdjustmentsReport'>;

// A report is read only with a valid period: enabled says so. Changing a
// filter keeps the last report on screen until the new one arrives.
const options = { placeholderData: keepPreviousData };

/** The period's sales by group (reports:read). */
export function useSalesReport(query: SalesQuery, enabled = true) {
  return useQuery({
    queryKey: reportsKeys.report('sales', query),
    queryFn: async () => unwrap(await api.GET('/api/v1/reports/sales', { params: { query } })),
    enabled,
    ...options,
  });
}

/** A page of the products sold, with their variants (costs only for whoever sees them). */
export function useProductsReport(query: ProductsQuery, enabled = true) {
  return useQuery({
    queryKey: reportsKeys.report('products', query),
    queryFn: async () => unwrap(await api.GET('/api/v1/reports/products', { params: { query } })),
    enabled,
    ...options,
  });
}

/** A page of the period's sales with their profit (only for whoever sees costs). */
export function useSaleProfitsReport(query: SaleProfitsQuery, enabled = true) {
  return useQuery({
    queryKey: reportsKeys.report('sale_profits', query),
    queryFn: async () => unwrap(await api.GET('/api/v1/reports/sale-profits', { params: { query } })),
    enabled,
    ...options,
  });
}

/** A page of the stock not sold in the last days. */
export function useIdleStockReport(query: IdleStockQuery) {
  return useQuery({
    queryKey: reportsKeys.report('idle_stock', query),
    queryFn: async () => unwrap(await api.GET('/api/v1/reports/idle-stock', { params: { query } })),
    ...options,
  });
}

/** The period's cash closings by group. */
export function useCashClosingsReport(query: CashClosingsQuery, enabled = true) {
  return useQuery({
    queryKey: reportsKeys.report('cash_closings', query),
    queryFn: async () => unwrap(await api.GET('/api/v1/reports/cash-closings', { params: { query } })),
    enabled,
    ...options,
  });
}

/** What each user did in the period, in total and by bucket. */
export function useTeamControlReport(query: TeamControlQuery, enabled = true) {
  return useQuery({
    queryKey: reportsKeys.report('team_control', query),
    queryFn: async () => unwrap(await api.GET('/api/v1/reports/team-control', { params: { query } })),
    enabled,
    ...options,
  });
}

/** The period's stock adjustments by user or by variant. */
export function useStockAdjustmentsReport(query: StockAdjustmentsQuery, enabled = true) {
  return useQuery({
    queryKey: reportsKeys.report('stock_adjustments', query),
    queryFn: async () => unwrap(await api.GET('/api/v1/reports/stock-adjustments', { params: { query } })),
    enabled,
    ...options,
  });
}
