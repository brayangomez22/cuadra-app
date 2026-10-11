import { useAuth } from './useAuth';

/**
 * Whether the signed-in user's role has a permission (the x-permission names of
 * api/openapi.yaml, as GET /me returns them). Only for showing or hiding
 * things: the API checks the permission on every request.
 */
export function useCan() {
  const auth = useAuth();
  const permissions = auth.status === 'authenticated' ? auth.user.permissions : [];
  return (permission: string) => permissions.includes(permission);
}
