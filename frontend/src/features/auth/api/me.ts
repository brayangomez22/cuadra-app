import { useQuery } from '@tanstack/react-query';
import { api } from '@/shared/api/client';
import { meQueryKey } from './keys';
import type { Me } from './types';

// fetchMe asks the API who is signed in. Without an access token (after a reload)
// the 401 makes authFetch renew the session with the refresh cookie and retry;
// if that fails too, there is no session.
async function fetchMe(): Promise<Me | null> {
  const { data, response } = await api.GET('/api/v1/me');
  if (data) return data;
  if (response.status === 401) return null;
  throw new Error(`GET /api/v1/me answered ${String(response.status)}`);
}

/** The signed-in user (null without a session), asked once per session. */
export function useMeQuery() {
  return useQuery({ queryKey: meQueryKey, queryFn: fetchMe, staleTime: Infinity, retry: false });
}
