import { salesRulesBody, salesRulesInput, salesRulesSchema } from './salesRulesSchema';

describe('salesRulesSchema', () => {
  it.each([
    { business: 'droguería', rules: { max_discount: '0', require_cash_session: true }, text: '0' },
    { business: 'almacén de ropa', rules: { max_discount: '0.15', require_cash_session: false }, text: '15' },
  ])('ida y vuelta de las reglas de venta ($business)', ({ rules, text }) => {
    const input = salesRulesInput(rules);
    expect(input.maxDiscount).toBe(text);

    expect(salesRulesBody(salesRulesSchema.parse(input))).toEqual(rules);
  });

  it('rechaza un descuento de más de 100 %', () => {
    const result = salesRulesSchema.safeParse({ maxDiscount: '150', requireCashSession: true });
    expect(result.error?.issues[0]?.message).toBe('El descuento va de 0 a 100 %');
  });
});
