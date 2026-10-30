import { pricePreview } from './pricePreview';

describe('pricePreview', () => {
  it.each([
    [
      'ferretería: cemento con IVA del 19 %',
      { price: '32000', cost: '28000', taxRate: '0.19' },
      { withTax: '38080.00', margin: '12.50' },
    ],
    [
      'ropa: camiseta, precio con coma decimal',
      { price: '25000,00', cost: '15000', taxRate: '0.19' },
      { withTax: '29750.00', margin: '40.00' },
    ],
    [
      'droguería: medicamento excluido de IVA',
      { price: '8500', cost: '6000', taxRate: '0' },
      { withTax: '8500.00', margin: '29.4118' },
    ],
    [
      'un precio sin IVA que da un precio redondo',
      { price: '8403,3613', cost: '0', taxRate: '0.19' },
      { withTax: '10000.00', margin: '100.00' },
    ],
    [
      'vende por debajo del costo',
      { price: '10000', cost: '12000', taxRate: '0.19' },
      { withTax: '11900.00', margin: '-20.00' },
    ],
    [
      'sin costo escrito todavía: solo el precio con IVA',
      { price: '10000', cost: '', taxRate: '0.19' },
      { withTax: '11900.00' },
    ],
  ])('%s', (_, input, expected) => {
    expect(pricePreview(input)).toEqual(expected);
  });

  it.each([
    ['sin precio', ''],
    ['un precio en cero', '0'],
    ['un precio que no es un número', 'veinte mil'],
  ])('%s no muestra nada', (_, price) => {
    expect(pricePreview({ price, cost: '8000', taxRate: '0.19' })).toBeUndefined();
  });
});
