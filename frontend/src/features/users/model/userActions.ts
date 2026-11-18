import type { User } from '../api/types';

type Role = User['role'];

/** What the signed-in manager may do to a user of the list. */
export type UserActions = { changeRole: boolean; resetPassword: boolean; deactivate: boolean; activate: boolean };

/**
 * The same rules the API enforces (owner_only, self_deactivation): only an
 * owner touches an owner, and nobody deactivates themselves. The last owner
 * is the API's to refuse (last_owner): the list may be stale.
 */
export function userActions(me: { id: string; role: Role }, user: User): UserActions {
  const manages = user.role !== 'owner' || me.role === 'owner';
  const self = me.id === user.id;
  return {
    changeRole: manages,
    resetPassword: manages && !self,
    deactivate: manages && !self && user.active,
    activate: manages && !user.active,
  };
}

/** The roles me may give: only an owner names owners. */
export function assignableRoles(me: { role: Role }): Role[] {
  const all: Role[] = ['owner', 'admin', 'cashier', 'warehouse'];
  return me.role === 'owner' ? all : all.filter((role) => role !== 'owner');
}
