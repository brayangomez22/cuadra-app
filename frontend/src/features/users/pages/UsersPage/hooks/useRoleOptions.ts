import { roleLabels, useCurrentUser } from '@/features/auth';
import type { SelectOption } from '@/shared/ui';
import { assignableRoles } from '../../../model/userActions';

export type RoleOption = SelectOption;

/** The roles the signed-in manager may give, as Select options. */
export function useRoleOptions(): RoleOption[] {
  const user = useCurrentUser();
  if (!user) return [];
  return assignableRoles(user).map((role) => ({ value: role, label: roleLabels[role] }));
}
