import { useAuth } from './useAuth';

/**
 * Whether a feature flag is on for the tenant ('feedback', 'analytics'...),
 * as GET /me returns them. A flag is a temporary switch of the Cuadra team,
 * not a module the business turns on (useModule).
 */
export function useFlag(key: string) {
  const auth = useAuth();
  return auth.status === 'authenticated' && auth.user.flags.includes(key);
}
