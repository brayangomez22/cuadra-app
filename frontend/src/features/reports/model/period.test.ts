import { defaultPeriod, periodProblem } from './period';

describe('defaultPeriod', () => {
  it.each([
    ['a mitad de mes, desde el primero del mes', '2026-11-19', { from: '2026-11-01', to: '2026-11-19' }],
    ['el primero del mes, solo ese día', '2026-12-01', { from: '2026-12-01', to: '2026-12-01' }],
  ])('%s', (_, today, expected) => {
    expect(defaultPeriod(today)).toEqual(expected);
  });
});

describe('periodProblem', () => {
  it.each([
    { name: 'un solo día está bien', from: '2026-11-03', to: '2026-11-03', expected: null },
    { name: 'un año bisiesto completo está bien', from: '2028-01-01', to: '2028-12-31', expected: null },
    {
      name: 'el último día antes del primero',
      from: '2026-11-04',
      to: '2026-11-03',
      expected: 'El último día debe ser igual o posterior al primero.',
    },
    {
      name: 'más de un año',
      from: '2026-01-01',
      to: '2027-01-02',
      expected: 'El periodo puede ser de un año como máximo.',
    },
    { name: 'un día vacío', from: '', to: '2026-11-03', expected: 'Elija el primer y el último día.' },
  ])('$name', ({ from, to, expected }) => {
    expect(periodProblem(from, to)).toBe(expected);
  });
});
