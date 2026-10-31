import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { unwrap } from '@/shared/api/errors';
import { salesKeys } from './keys';
import type { CashMovementInput, CloseCashSessionInput, OpenCashSessionInput } from './types';

/** The user's open cash session, if any, and whether the tenant requires one to sell. */
export function useCurrentCashSession() {
  return useQuery({
    queryKey: salesKeys.currentCash,
    queryFn: async () => unwrap(await api.GET('/api/v1/cash-sessions/current')),
  });
}

/**
 * Reads the user's open cash session again: the POS calls it when the API
 * says the drawer is gone, and the closing screen once the drawer closed.
 */
export function useRefreshCurrentCash() {
  const queryClient = useQueryClient();
  return () => void queryClient.invalidateQueries({ queryKey: salesKeys.currentCash });
}

export function useCashSession(id: string, { enabled = true } = {}) {
  return useQuery({
    enabled,
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
