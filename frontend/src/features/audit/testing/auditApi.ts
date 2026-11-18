import type { components } from '@/shared/api/schema';
import { aUser, fakeApi, json, signedIn, type Me } from '@/test/fakeApi';

type AuditEvent = components['schemas']['AuditEvent'];
type User = components['schemas']['User'];

export const cashierUser: User = {
  id: '01920000-0000-7000-8000-000000000201',
  name: 'Luis Pérez',
  email: 'luis@eltornillo.co',
  role: 'cashier',
  active: true,
  created_at: '2026-10-01T15:00:00Z',
};

export const saleId = '01920000-0000-7000-8000-000000000901';

export const voided: AuditEvent = {
  id: '01920000-0000-7000-8000-000000000801',
  action: 'sale.voided',
  user_id: cashierUser.id,
  role: 'cashier',
  entity_type: 'sale',
  entity_id: saleId,
  occurred_at: '2026-11-18T15:30:00Z',
  details: { total: '185000.00' },
};

export const priceChanged: AuditEvent = {
  id: '01920000-0000-7000-8000-000000000802',
  action: 'variant.price_changed',
  user_id: aUser().id,
  role: 'owner',
  entity_type: 'variant',
  entity_id: '01920000-0000-7000-8000-000000000902',
  occurred_at: '2026-11-18T14:00:00Z',
  details: { old_price: '4500.00', new_price: '4800.00' },
};

/** The audit screen of the owner, with two events and the users that name them. */
export function auditApi(routes: Parameters<typeof fakeApi>[0] = {}, me: Me = aUser()) {
  return fakeApi({
    ...signedIn(me),
    'GET /api/v1/users': json(200, [cashierUser]),
    'GET /api/v1/audit-events': json(200, { items: [voided, priceChanged], total: 2 }),
    ...routes,
  });
}
