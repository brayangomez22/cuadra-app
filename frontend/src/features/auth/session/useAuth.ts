import { useMeQuery } from '../api/me';
import { logout, useStartSession } from '../api/session';
import type { Me } from '../api/types';

export type AuthState =
  | { status: 'loading' }
  // The session could not be checked (server error or no connection).
  | { status: 'error'; retry: () => void }
  | { status: 'anonymous' }
  | { status: 'authenticated'; user: Me };

function authState(me: ReturnType<typeof useMeQuery>): AuthState {
  if (me.isPending) return { status: 'loading' };
  if (me.isError) return { status: 'error', retry: () => void me.refetch() };
  if (me.data) return { status: 'authenticated', user: me.data };
  return { status: 'anonymous' };
}

/**
 * The session: who is signed in, startSession for the token of a login or
 * sign-up, and logout.
 */
export function useAuth() {
  const me = useMeQuery();
  const startSession = useStartSession();
  return { ...authState(me), startSession, logout };
}
