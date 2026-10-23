import { noDiscount, tenPercent, unlimited } from '../../testing/cart';
import { readDiscount } from './discount';

describe('readDiscount', () => {
  it('sin texto no hay descuento, aunque el tope sea 0', () => {
    expect(readDiscount('', noDiscount)).toEqual({ fraction: 0n });
    expect(readDiscount('0', noDiscount)).toEqual({ fraction: 0n });
  });

  it('convierte el porcentaje en una fracción con 4 decimales', () => {
    expect(readDiscount('10', tenPercent)).toEqual({ fraction: 1000n });
    expect(readDiscount('2,5', unlimited)).toEqual({ fraction: 250n });
  });

  it('sin tope acepta el 100 %', () => {
    expect(readDiscount('100', unlimited)).toEqual({ fraction: 10000n });
  });

  it.each([
    ['1', noDiscount, 'No puede dar descuentos'],
    ['10,5', tenPercent, 'Máximo 10 %'],
    ['101', unlimited, 'El descuento va de 0 a 100 %'],
    ['2,555', unlimited, 'Use máximo 2 decimales'],
    ['diez', unlimited, 'Escriba un porcentaje, por ejemplo 5 o 2,5'],
  ])('rechaza «%s» con el tope dado', (text, limit, error) => {
    expect(readDiscount(text, limit)).toEqual({ error });
  });
});
