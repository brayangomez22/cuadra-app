import { proposedDueDate } from './proposedDueDate';

describe('proposedDueDate', () => {
  it.each([
    {
      name: 'suma el plazo del proveedor a la fecha de la factura',
      issuedOn: '2026-10-05',
      days: 30,
      due: '2026-11-04',
    },
    { name: 'cruza el fin de mes', issuedOn: '2026-01-31', days: 1, due: '2026-02-01' },
    { name: 'cruza el fin de año', issuedOn: '2026-12-20', days: 15, due: '2027-01-04' },
    { name: 'respeta el 29 de febrero', issuedOn: '2028-02-28', days: 1, due: '2028-02-29' },
    { name: 'sin plazo vence el mismo día', issuedOn: '2026-10-05', days: 0, due: '2026-10-05' },
  ])('$name', ({ issuedOn, days, due }) => {
    expect(proposedDueDate(issuedOn, days)).toBe(due);
  });

  it('sin fecha de factura no propone nada', () => {
    expect(proposedDueDate('', 30)).toBe('');
  });
});
