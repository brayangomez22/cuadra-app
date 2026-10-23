import { useQueryClient } from '@tanstack/react-query';
import { useCallback, useEffect } from 'react';
import { api, session } from '@/shared/api/client';
import { meQueryKey } from './keys';
import type { AuthSession, Me } from './types';

/** Keeps the token of a login or sign-up and the user that came with it. */
export function useStartSession() {
  const queryClient = useQueryClient();
  return useCallback(
    (auth: AuthSession) => {
      session.setAccessToken(auth.access_token);
      queryClient.setQueryData<Me | null>(meQueryKey, auth.user);
    },
    [queryClient],
  );
}

/**
 * Revokes the refresh cookie and ends the session here even if the API cannot
 * be reached: the user asked to leave this device.
 */
export async function logout() {
  try {
    await api.POST('/api/v1/auth/logout', { params: {} });
  } catch {
    // Without connection the cookie stays valid until it expires; nothing else to do.
  } finally {
    session.end();
  }
}

/**
 * When the session ends (logout, or a refresh that failed), drops the cached
 * data of the previous user and marks that there is no user. It sets the user
 * to null instead of removing the query: removing it would refetch /me, fail
 * and end the session again.
 */
export function useClearCacheOnSessionEnd() {
  const queryClient = useQueryClient();
  useEffect(
    () =>
      session.onEnd(() => {
        queryClient.removeQueries({ predicate: (query) => query.queryKey[0] !== meQueryKey[0] });
        queryClient.setQueryData<Me | null>(meQueryKey, null);
      }),
    [queryClient],
  );
}
