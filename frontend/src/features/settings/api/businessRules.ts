import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { unwrap } from '@/shared/api/errors';
import type { components } from '@/shared/api/schema';
import { settingsKeys } from './keys';

export type SalesRules = components['schemas']['SalesSettings'];
export type StockRules = components['schemas']['InventorySettings'];

/** The cashier's discount limit and whether selling needs an open cash session. */
export function useSalesRules() {
  return useQuery({
    queryKey: settingsKeys.salesRules,
    queryFn: async () => unwrap(await api.GET('/api/v1/sales/settings')),
  });
}

/** Changes the sales rules; the answer replaces the cached ones. */
export function useUpdateSalesRules() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: components['schemas']['SalesSettingsUpdate']) =>
      unwrap(await api.PATCH('/api/v1/sales/settings', { body })),
    onSuccess: (rules) => queryClient.setQueryData(settingsKeys.salesRules, rules),
  });
}

/** Whether sales, adjustments and transfers may take the stock below zero. */
export function useStockRules() {
  return useQuery({
    queryKey: settingsKeys.stockRules,
    queryFn: async () => unwrap(await api.GET('/api/v1/inventory/settings')),
  });
}

/** Changes the stock rules; the answer replaces the cached ones. */
export function useUpdateStockRules() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: components['schemas']['InventorySettingsUpdate']) =>
      unwrap(await api.PATCH('/api/v1/inventory/settings', { body })),
    onSuccess: (rules) => queryClient.setQueryData(settingsKeys.stockRules, rules),
  });
}
