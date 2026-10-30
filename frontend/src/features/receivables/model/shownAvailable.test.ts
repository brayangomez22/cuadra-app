import { shownAvailable } from './shownAvailable';

describe('shownAvailable', () => {
  it.each([
    ['un saldo con 50 centavos: la resta de lo que se ve cuadra', '700000.00', '356889.50', '343110.00'],
    ['sin centavos es lo que dice la API', '1500000.00', '350000.00', '1150000.00'],
    ['por debajo de medio peso no cambia', '100000.00', '20000.49', '80000.00'],
    ['un cupo con centavos', '100000.50', '20000.00', '80001.00'],
    ['si debe más que el cupo no muestra un negativo', '100000.00', '150000.50', '0.00'],
  ])('%s', (...[, credit_limit, balance, expected]) => {
    expect(shownAvailable({ credit_limit, balance })).toBe(expected);
  });
});
