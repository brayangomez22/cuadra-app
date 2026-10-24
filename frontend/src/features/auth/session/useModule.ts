import { useAuth } from './useAuth';

/**
 * Whether the tenant has a module active (its stable id: 'variants',
 * 'unit_conversions'...), as GET /me returns them. Only for showing or hiding
 * things: the API answers 404 on the routes of an inactive module.
 */
export function useModule(id: string) {
  const auth = useAuth();
  return auth.status === 'authenticated' && auth.user.modules.includes(id);
}
