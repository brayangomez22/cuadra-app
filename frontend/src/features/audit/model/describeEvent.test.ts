import type { AuditEvent } from '../api/types';
import { actionLabels } from './actions';
import { describeEvent } from './describeEvent';

const event = (action: AuditEvent['action'], details: Record<string, string>): AuditEvent => ({
  id: 'e-1',
  action,
  user_id: 'u-1',
  role: 'cashier',
  entity_type: 'sale',
  occurred_at: '2026-11-18T15:00:00Z',
  details,
});

describe('describeEvent', () => {
  it.each([
    ['ferretería: anula una venta', event('sale.voided', { total: '185000.00' }), 'Venta de $ 185.000'],
    [
      'almacén de ropa: descuento en una venta',
      event('sale.discount_applied', { discount: '12000.00', gross: '120000.00' }),
      '$ 12.000 de descuento sobre $ 120.000',
    ],
    [
      'ferretería: cambia un precio',
      event('variant.price_changed', { old_price: '4500.00', new_price: '4800.00' }),
      'De $ 4.500 a $ 4.800',
    ],
    [
      'almacén de ropa: ajusta el stock de una talla',
      event('stock.adjusted', { quantity: '-2', source: 'adjustment' }),
      '-2 en un ajuste manual',
    ],
    [
      'ferretería: aprueba un conteo con diferencia',
      event('stock.adjusted', { quantity: '3', source: 'stock_count' }),
      '+3 en una toma física',
    ],
    [
      'cierre de caja con faltante',
      event('cash.closed_with_difference', { expected: '350000.00', counted: '340000.00', difference: '-10000.00' }),
      'Faltaron $ 10.000 (contó $ 340.000 de $ 350.000)',
    ],
    [
      'cierre de caja con sobrante',
      event('cash.closed_with_difference', { expected: '100000.00', counted: '100500.00', difference: '500.00' }),
      'Sobraron $ 500 (contó $ 100.500 de $ 100.000)',
    ],
    [
      'cambia el rol de un usuario',
      event('user.role_changed', { old_role: 'cashier', new_role: 'admin' }),
      'De Cajero a Administrador',
    ],
    ['invita a un bodeguero', event('user.invited', { role: 'warehouse' }), 'Como Bodeguero'],
    [
      'sube el cupo de un cliente',
      event('credit.terms_changed', {
        old_limit: '100000.00',
        new_limit: '500000.00',
        old_term_days: '15',
        new_term_days: '30',
      }),
      'Cupo de $ 100.000 a $ 500.000; plazo de 15 a 30 días',
    ],
    [
      'cambia el IVA por defecto',
      event('tenant.settings_changed', { setting: 'default_tax_rate', old: '0.19', new: '0.05' }),
      'IVA por defecto: de 0.19 a 0.05',
    ],
    ['activa un módulo', event('module.enabled', { module: 'variants' }), 'variants'],
    ['desactiva un usuario', event('user.deactivated', {}), ''],
  ])('%s', (_, e, want) => {
    expect(describeEvent(e).replace(/\u00a0/g, ' ')).toBe(want);
  });
});

describe('actionLabels', () => {
  it('nombra en español cada acción del catálogo', () => {
    expect(actionLabels['sale.voided']).toBe('Anuló una venta');
    expect(Object.values(actionLabels).every((label) => label.length > 0)).toBe(true);
  });
});
