import type { TeamControlReport } from '../../api/types';
import { weekAlerts } from './weekAlerts';

const zero = {
  voids: 0,
  voids_total: '0.00',
  discounts: 0,
  discounts_amount: '0.00',
  price_changes: 0,
  adjustments: 0,
  cash_differences: 0,
  cash_shortage: '0.00',
  cash_overage: '0.00',
};
const ana = { user_id: 'ana', name: 'Ana (ferretería)' };
const luis = { user_id: 'luis', name: 'Luis (almacén de ropa)' };
const week = (start: string, user: { user_id: string }, figures: Partial<typeof zero>) => ({
  ...zero,
  ...figures,
  user_id: user.user_id,
  bucket_start: start,
});

// Luis voids one sale a week and five this week, and his drawer came up
// short this week; Ana discounts the same every week.
const report: TeamControlReport = {
  period: { from: '2026-11-02', to: '2026-11-19' },
  bucket: 'week',
  users: [
    { ...zero, ...ana, discounts: 6, discounts_amount: '30000.00' },
    { ...zero, ...luis, voids: 7, cash_differences: 1, cash_shortage: '8000.00' },
  ],
  series: [
    week('2026-11-02', luis, { voids: 1 }),
    week('2026-11-02', ana, { discounts: 2, discounts_amount: '10000.00' }),
    week('2026-11-09', luis, { voids: 1 }),
    week('2026-11-09', ana, { discounts: 2, discounts_amount: '10000.00' }),
    week('2026-11-16', luis, { voids: 5, cash_differences: 1, cash_shortage: '8000.00' }),
    week('2026-11-16', ana, { discounts: 2, discounts_amount: '10000.00' }),
  ],
};

describe('weekAlerts', () => {
  it('alerta lo de esta semana que pasa del doble del promedio de esa persona', () => {
    expect(weekAlerts(report, '2026-11-16')).toEqual([
      { userId: 'luis', name: 'Luis (almacén de ropa)', metric: 'voids', value: '5' },
      { userId: 'luis', name: 'Luis (almacén de ropa)', metric: 'cash_shortage', value: '8000.00' },
    ]);
  });

  it('no alerta a quien está en su promedio ni las cifras en cero', () => {
    const alerts = weekAlerts(report, '2026-11-16');
    expect(alerts.some((alert) => alert.userId === 'ana')).toBe(false);
  });

  it('lo que se salió de lo normal en otra semana no es una alerta de esta', () => {
    expect(weekAlerts(report, '2026-11-09')).toEqual([]);
  });

  it('sin actividad en el periodo no hay alertas', () => {
    expect(weekAlerts({ ...report, users: [], series: [] }, '2026-11-16')).toEqual([]);
  });
});
