import type { Me } from '../api/types';
import { useAuth } from './useAuth';

/** The signed-in user, or null outside a session (the screens behind RequireAuth always have one). */
export function useCurrentUser(): Me | null {
  const auth = useAuth();
  return auth.status === 'authenticated' ? auth.user : null;
}
