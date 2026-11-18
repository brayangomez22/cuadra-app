import type { User } from '../api/types';
import { assignableRoles, userActions } from './userActions';

const user = (over: Partial<User>): User => ({
  id: 'u-1',
  name: 'Luis Pérez',
  email: 'luis@eltornillo.co',
  role: 'cashier',
  active: true,
  created_at: '2026-10-01T15:00:00Z',
  ...over,
});

describe('userActions', () => {
  const owner = { id: 'me', role: 'owner' as const };
  const admin = { id: 'me', role: 'admin' as const };

  it.each([
    {
      name: 'ferretería: el dueño gestiona a su cajero',
      me: owner,
      target: user({}),
      want: { changeRole: true, resetPassword: true, deactivate: true, activate: false },
    },
    {
      name: 'almacén de ropa: el administrador reactiva a un bodeguero',
      me: admin,
      target: user({ role: 'warehouse', active: false }),
      want: { changeRole: true, resetPassword: true, deactivate: false, activate: true },
    },
    {
      name: 'el administrador no toca al dueño',
      me: admin,
      target: user({ role: 'owner' }),
      want: { changeRole: false, resetPassword: false, deactivate: false, activate: false },
    },
    {
      name: 'el dueño no se desactiva ni se envía un enlace a sí mismo',
      me: owner,
      target: user({ id: 'me', role: 'owner' }),
      want: { changeRole: true, resetPassword: false, deactivate: false, activate: false },
    },
  ])('$name', ({ me, target, want }) => {
    expect(userActions(me, target)).toEqual(want);
  });
});

describe('assignableRoles', () => {
  it('solo el dueño nombra dueños', () => {
    expect(assignableRoles({ role: 'owner' })).toContain('owner');
    expect(assignableRoles({ role: 'admin' })).toEqual(['admin', 'cashier', 'warehouse']);
  });
});
