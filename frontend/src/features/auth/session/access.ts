import { useAuth } from './useAuth';

/** What the signed-in user may see: the permissions and active modules of GET /me. */
export type Access = {
  permissions: readonly string[];
  modules: readonly string[];
};

/**
 * Gated is anything the UI shows only to some users: a menu item, a section
 * or a field. permission is an x-permission of api/openapi.yaml; module, the
 * id of an activable module (variants, lots_expiry...).
 */
export type Gated = {
  permission?: string;
  module?: string;
};

/** Whether access lets the user see item. Only for showing or hiding: the API checks both. */
export function canSee(item: Gated, access: Access) {
  return (
    (!item.permission || access.permissions.includes(item.permission)) &&
    (!item.module || access.modules.includes(item.module))
  );
}

/** The items access lets the user see, in their order. */
export function visibleItems<T extends Gated>(items: readonly T[], access: Access): T[] {
  return items.filter((item) => canSee(item, access));
}

/** The signed-in user's access; nothing while there is no session. */
export function useAccess(): Access {
  const auth = useAuth();
  if (auth.status !== 'authenticated') return { permissions: [], modules: [] };
  return { permissions: auth.user.permissions, modules: auth.user.modules };
}
