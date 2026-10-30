import { overdueSummary } from './overdue';

const today = '2026-10-28';
const owed = (balance: string, due_date: string) => ({ balance, due_date });

describe('overdueSummary', () => {
  it('suma lo vencido y dice cuántos días lleva la cuenta más antigua', () => {
    const receivables = [
      owed('120000.50', '2026-06-07'),
      owed('80000', '2026-10-01'),
      owed('50000', '2026-10-28'),
      owed('30000', '2026-11-27'),
    ];

    expect(overdueSummary(receivables, today)).toEqual({ count: 2, balance: '200000.50', oldestDays: 143 });
  });

  it('sin cuentas vencidas no hay mora, aunque una venza hoy', () => {
    expect(overdueSummary([owed('50000', '2026-10-28'), owed('9000', '2026-11-02')], today)).toBeUndefined();
  });

  it('sin cuentas abiertas no hay mora', () => {
    expect(overdueSummary([], today)).toBeUndefined();
  });
});
