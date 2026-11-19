import type { TeamControlReport } from '../api/types';
import { trendTable } from './teamTrend';

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

// Luis voids one sale most weeks and five in the third; Ana discounts steadily.
const report: TeamControlReport = {
  period: { from: '2026-11-02', to: '2026-11-22' },
  bucket: 'week',
  users: [
    { ...zero, ...ana, discounts: 6, discounts_amount: '30000.00' },
    { ...zero, ...luis, voids: 7 },
  ],
  series: [
    { ...zero, user_id: 'luis', bucket_start: '2026-11-02', voids: 1 },
    { ...zero, user_id: 'ana', bucket_start: '2026-11-02', discounts: 2, discounts_amount: '10000.00' },
    { ...zero, user_id: 'luis', bucket_start: '2026-11-09', voids: 1 },
    { ...zero, user_id: 'ana', bucket_start: '2026-11-09', discounts: 2, discounts_amount: '10000.00' },
    { ...zero, user_id: 'luis', bucket_start: '2026-11-16', voids: 5 },
    { ...zero, user_id: 'ana', bucket_start: '2026-11-16', discounts: 2, discounts_amount: '10000.00' },
  ],
};

describe('trendTable', () => {
  it('pone a cada usuario en una fila y cada semana en una columna, en orden', () => {
    const table = trendTable(report, 'voids');
    expect(table.buckets).toEqual(['2026-11-02', '2026-11-09', '2026-11-16']);
    expect(table.rows.map((row) => row.name)).toEqual(['Ana (ferretería)', 'Luis (almacén de ropa)']);
    expect(table.rows[1]?.cells.map((cell) => cell.value)).toEqual(['1', '1', '5']);
  });

  it('marca la semana que se sale de lo normal de ese usuario (más del doble de su promedio)', () => {
    const luisRow = trendTable(report, 'voids').rows[1];
    expect(luisRow?.cells.map((cell) => cell.unusual)).toEqual([false, false, true]);
  });

  it('lo que es parejo no se marca, aunque sea alto', () => {
    const anaRow = trendTable(report, 'discounts_amount').rows[0];
    expect(anaRow?.cells.map((cell) => cell.value)).toEqual(['10000.00', '10000.00', '10000.00']);
    expect(anaRow?.cells.some((cell) => cell.unusual)).toBe(false);
  });

  it('una semana sin datos de un usuario cuenta como cero', () => {
    const shortage = trendTable(
      {
        ...report,
        series: [...report.series, { ...zero, user_id: 'ana', bucket_start: '2026-11-23', cash_shortage: '5000.00' }],
      },
      'cash_shortage',
    );
    expect(shortage.rows[1]?.cells.at(-1)).toEqual({ bucket: '2026-11-23', value: '0', unusual: false });
    expect(shortage.rows[0]?.cells.at(-1)?.unusual).toBe(true);
  });
});
