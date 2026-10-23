import { aGarment, aProduct, boxOf12, ids, unitOf, withUnits } from '@/test/fixtures';
import { noDiscount, tenPercent } from '../../testing/cart';
import { cartItem } from './cartItem';
import { lineTotals } from './lineTotals';

describe('lineTotals', () => {
  it('una línea con un campo inválido no tiene montos ni cuerpo', () => {
    const line = { item: cartItem(aProduct()), quantity: '0', discount: '15' };
    expect(lineTotals(line, tenPercent)).toEqual({
      line,
      quantityError: 'La cantidad debe ser mayor que cero',
      discountError: 'Máximo 10 %',
    });
  });

  it('vende en la unidad alternativa a su precio y envía unit_id', () => {
    const line = {
      item: cartItem(withUnits(aProduct(), [boxOf12])),
      quantity: '1',
      discount: '',
      unitId: unitOf('caja').id,
    };
    const totals = lineTotals(line, noDiscount);
    expect(totals.amounts).toEqual({
      gross: '110000.00',
      discount: '0.00',
      base: '110000.00',
      tax: '20900.00',
      total: '130900.00',
    });
    expect(totals.input).toEqual({ variant_id: ids.pipe, quantity: '1', unit_id: unitOf('caja').id });
  });

  it('el descuento viaja en el cuerpo solo si es mayor que cero', () => {
    const garment = cartItem(aGarment());
    expect(lineTotals({ item: garment, quantity: '2', discount: '0' }, tenPercent).input).not.toHaveProperty(
      'discount',
    );
    // 10 % de 2 × 25.000 = 5.000 antes de IVA.
    expect(lineTotals({ item: garment, quantity: '2', discount: '10' }, tenPercent).input).toMatchObject({
      discount: '5000.00',
    });
  });
});
