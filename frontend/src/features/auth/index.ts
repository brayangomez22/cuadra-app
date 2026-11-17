// The public API of auth: the session, the route guards and what the user may see.
export type { Me } from './api/types';
export { meQueryKey } from './api/keys';
export { GuestOnly } from './guards/GuestOnly/GuestOnly';
export { RequireAuth } from './guards/RequireAuth/RequireAuth';
export { RequireOperator } from './guards/RequireOperator/RequireOperator';
export { RequirePermission } from './guards/RequirePermission/RequirePermission';
export { canSee, useAccess, visibleItems, type Access, type Gated } from './session/access';
export { AuthProvider } from './session/AuthProvider';
export { useAuth } from './session/useAuth';
export { useCan } from './session/useCan';
export { useFlag } from './session/useFlag';
export { useModule } from './session/useModule';
export { roleLabels } from './model/roles';
