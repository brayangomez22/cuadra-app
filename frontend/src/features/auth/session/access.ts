import { useAuth } from './useAuth';

/**
 * What the signed-in user may see: the permissions, active modules and feature
 * flags of GET /me, and whether the user operates the platform.
 */
export type Access = {
  permissions: readonly string[];
  modules: readonly string[];
  flags?: readonly string[];
  operator?: boolean;
};

/**
 * Gated is anything the UI shows only to some users: a menu item, a section
 * or a field. permission is an x-permission of api/openapi.yaml; module, the
 * id of an activable module (variants, lots_expiry...); flag, a feature flag
 * being released; operator, only for the Cuadra team.
 */
export type Gated = {
  permission?: string;
  module?: string;
  flag?: string;
  operator?: boolean;
};

function allowedByRole(item: Gated, access: Access) {
  return (
    (!item.permission || access.permissions.includes(item.permission)) &&
    (!item.module || access.modules.includes(item.module))
  );
}

function allowedByPlatform(item: Gated, access: Access) {
  return (!item.flag || (access.flags ?? []).includes(item.flag)) && (!item.operator || access.operator === true);
}

/** Whether access lets the user see item. Only for showing or hiding: the API checks every request. */
export function canSee(item: Gated, access: Access) {
  return allowedByRole(item, access) && allowedByPlatform(item, access);
}

/** The items access lets the user see, in their order. */
export function visibleItems<T extends Gated>(items: readonly T[], access: Access): T[] {
  return items.filter((item) => canSee(item, access));
}

/** The signed-in user's access; nothing while there is no session. */
export function useAccess(): Access {
  const auth = useAuth();
  if (auth.status !== 'authenticated') return { permissions: [], modules: [] };
  const { permissions, modules, flags, operator } = auth.user;
  return { permissions, modules, flags, operator };
}
