import { posMethods } from './paymentMethods';

describe('posMethods', () => {
  it('sin un cliente con cupo no ofrece crédito', () => {
    expect(posMethods(false)).toEqual(['cash', 'card', 'nequi', 'daviplata', 'transfer']);
  });

  it('con un cliente con cupo ofrece crédito al final', () => {
    expect(posMethods(true)).toEqual(['cash', 'card', 'nequi', 'daviplata', 'transfer', 'credit']);
  });
});
