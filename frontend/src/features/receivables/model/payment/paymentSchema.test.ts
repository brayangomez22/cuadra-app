import { paymentSchema } from './paymentSchema';

describe('paymentSchema', () => {
  const owes = paymentSchema('250000.00');

  it.each([
    ['100.000', '100000'],
    ['250000', '250000'],
    ['0,50', '0.50'],
  ])('acepta un abono de «%s»', (amount, expected) => {
    expect(owes.parse({ method: 'cash', amount })).toEqual({ method: 'cash', amount: expected });
  });

  it.each([
    ['250.001', 'El abono no puede pasar de lo que debe ($ 250.000)'],
    ['0', 'El valor debe ser mayor que cero'],
    ['', 'Escriba el valor del abono'],
  ])('rechaza un abono de «%s»', (amount, message) => {
    const issue = owes.safeParse({ method: 'nequi', amount }).error?.issues[0];
    expect(issue?.path).toEqual(['amount']);
    expect(issue?.message.replace(/\s/g, ' ')).toBe(message);
  });
});
