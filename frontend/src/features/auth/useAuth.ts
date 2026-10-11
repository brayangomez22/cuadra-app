import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useCallback } from 'react';
import { api, session } from '@/shared/api/client';
import type { components } from '@/shared/api/schema';

export type Me = components['schemas']['Me'];
export type AuthSession = components['schemas']['AuthSession'];

export type AuthState =
  | { status: 'loading' }
  // The session could not be checked (server error or no connection).
  | { status: 'error'; retry: () => void }
  | { status: 'anonymous' }
  | { status: 'authenticated'; user: Me };

// The current user: null when there is no session.
export const meQueryKey = ['me'] as const;

// fetchMe asks the API who is signed in. Without an access token (after a reload)
// the 401 makes authFetch renew the session with the refresh cookie and retry;
// if that fails too, there is no session.
async function fetchMe(): Promise<Me | null> {
  const { data, response } = await api.GET('/api/v1/me');
  if (data) return data;
  if (response.status === 401) return null;
  throw new Error(`GET /api/v1/me answered ${response.status}`);
}

export function useAuth() {
  const queryClient = useQueryClient();
  const me = useQuery({ queryKey: meQueryKey, queryFn: fetchMe, staleTime: Infinity, retry: false });
  const { refetch } = me;

  // startSession keeps the token of a login or sign-up and the user that came with it.
  const startSession = useCallback(
    (auth: AuthSession) => {
      session.setAccessToken(auth.access_token);
      queryClient.setQueryData<Me | null>(meQueryKey, auth.user);
    },
    [queryClient],
  );

  // logout revokes the refresh cookie and ends the session here even if the API
  // cannot be reached: the user asked to leave this device.
  const logout = useCallback(async () => {
    try {
      await api.POST('/api/v1/auth/logout', { params: {} });
    } catch {
      // Without connection the cookie stays valid until it expires; nothing else to do.
    } finally {
      session.end();
    }
  }, []);

  let state: AuthState;
  if (me.isPending) state = { status: 'loading' };
  else if (me.isError) state = { status: 'error', retry: () => void refetch() };
  else if (me.data) state = { status: 'authenticated', user: me.data };
  else state = { status: 'anonymous' };

  return { ...state, startSession, logout };
}
