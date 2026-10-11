import { useQueryClient } from '@tanstack/react-query';
import { useEffect, type ReactNode } from 'react';
import { session } from '@/shared/api/client';
import { meQueryKey, type Me } from './useAuth';

// AuthProvider reacts to the end of the session (logout, or a refresh that
// failed): it drops the cached data of the previous user and marks that there
// is no user, so RequireAuth sends the page to the login. It sets the user to
// null instead of removing the query: removing it would refetch /me, fail and
// end the session again.
export function AuthProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient();

  useEffect(
    () =>
      session.onEnd(() => {
        queryClient.removeQueries({ predicate: (query) => query.queryKey[0] !== meQueryKey[0] });
        queryClient.setQueryData<Me | null>(meQueryKey, null);
      }),
    [queryClient],
  );

  return children;
}
