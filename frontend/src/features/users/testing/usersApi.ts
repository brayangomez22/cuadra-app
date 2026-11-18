import type { components } from '@/shared/api/schema';
import { aUser, fakeApi, json, signedIn, type Me } from '@/test/fakeApi';

type User = components['schemas']['User'];
type Invitation = components['schemas']['Invitation'];

const id = (n: number) => `01920000-0000-7000-8000-00000000${String(n).padStart(4, '0')}`;

/** The signed-in owner of aUser(), as GET /users lists them. */
export const owner: User = {
  id: aUser().id,
  name: 'Ana Gómez',
  email: 'ana@eltornillo.co',
  role: 'owner',
  active: true,
  created_at: '2026-10-01T15:00:00Z',
};

// A hardware store's cashier and a clothing store's warehouse keeper: the
// screen works the same for any business.
export const cashier: User = {
  ...owner,
  id: id(201),
  name: 'Luis Pérez',
  email: 'luis@eltornillo.co',
  role: 'cashier',
};
export const keeper: User = {
  ...owner,
  id: id(202),
  name: 'Marta Ríos',
  email: 'marta@modaandina.co',
  role: 'warehouse',
  active: false,
};

export const invitation: Invitation = {
  id: id(301),
  email: 'nuevo@eltornillo.co',
  name: 'Pedro Nuevo',
  role: 'cashier',
  status: 'pending',
  invited_by: owner.id,
  expires_at: '2026-11-25T15:00:00Z',
  created_at: '2026-11-18T15:00:00Z',
};

export const linkToken = '01920000-0000-7000-8000-0000000000aa.c2VjcmV0by1kZS1wcnVlYmEtcGFyYS1lbC1lbmxhY2U';

/** The users screen of `me`, with its users and open invitations. */
export function usersApi(routes: Parameters<typeof fakeApi>[0] = {}, me: Me = aUser()) {
  return fakeApi({
    ...signedIn(me),
    'GET /api/v1/users': json(200, [owner, cashier, keeper]),
    'GET /api/v1/invitations': json(200, [invitation]),
    ...routes,
  });
}
