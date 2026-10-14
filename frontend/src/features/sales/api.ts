import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { unwrap } from '@/shared/api/errors';
import type { components } from '@/shared/api/schema';

export type Sale = components['schemas']['Sale'];
export type ConfirmSaleInput = components['schemas']['ConfirmSaleInput'];
export type CashSession = components['schemas']['CashSession'];
export type CashMovement = components['schemas']['CashMovement'];
export type CashMovementInput = components['schemas']['CashMovementInput'];
export type CloseCashSessionInput = components['schemas']['CloseCashSessionInput'];
export type OpenCashSessionInput = components['schemas']['OpenCashSessionInput'];

export const salesKeys = {
  discountLimit: ['sales', 'discount-limit'] as const,
  currentCash: ['sales', 'cash-sessions', 'current'] as const,
  cashSession: (id: string) => ['sales', 'cash-sessions', id] as const,
};

/** How much the user may discount on a line. The API applies it again on checkout. */
export function useDiscountLimit() {
  return useQuery({
    queryKey: salesKeys.discountLimit,
    queryFn: async () => unwrap(await api.GET('/api/v1/sales/discount-limit')),
  });
}

export function useConfirmSale() {
  return useMutation({
    mutationFn: async (body: ConfirmSaleInput) => unwrap(await api.POST('/api/v1/sales', { body })),
  });
}

/** The user's open cash session, if any, and whether the tenant requires one to sell. */
export function useCurrentCashSession() {
  return useQuery({
    queryKey: salesKeys.currentCash,
    queryFn: async () => unwrap(await api.GET('/api/v1/cash-sessions/current')),
  });
}

export function useCashSession(id: string) {
  return useQuery({
    queryKey: salesKeys.cashSession(id),
    queryFn: async () =>
      unwrap(
        await api.GET('/api/v1/cash-sessions/{cashSessionId}', {
          params: { path: { cashSessionId: id } },
        }),
      ),
  });
}

export function useOpenCashSession() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: OpenCashSessionInput) => unwrap(await api.POST('/api/v1/cash-sessions', { body })),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: salesKeys.currentCash }),
  });
}

export function useRecordCashMovement(sessionId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: CashMovementInput) =>
      unwrap(
        await api.POST('/api/v1/cash-sessions/{cashSessionId}/movements', {
          params: { path: { cashSessionId: sessionId } },
          body,
        }),
      ),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: salesKeys.currentCash }),
  });
}

export function useCloseCashSession(sessionId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: CloseCashSessionInput) =>
      unwrap(
        await api.POST('/api/v1/cash-sessions/{cashSessionId}/close', {
          params: { path: { cashSessionId: sessionId } },
          body,
        }),
      ),
    // The caller reads the current session again: the closing screen must
    // first know it closed, or it would see no session and send to /caja.
    onSuccess: (session) => queryClient.setQueryData(salesKeys.cashSession(session.id), session),
  });
}
