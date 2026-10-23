import { readQuantity } from './quantity';

describe('readQuantity', () => {
  it.each([
    ['2,5', '2.5'],
    ['0,75', '0.75'],
    ['3', '3'],
  ])('acepta %s metros o kilos como cantidad fraccionaria', (text, value) => {
    expect(readQuantity(text)).toEqual({ value });
  });

  it.each([
    ['', 'Escriba la cantidad'],
    ['  ', 'Escriba la cantidad'],
    ['abc', 'Escriba un número, por ejemplo 2 o 2,5'],
    ['1.000', 'Escriba la cantidad sin puntos de miles: 1000'],
    ['1,23456', 'Use máximo 4 decimales'],
    ['0', 'La cantidad debe ser mayor que cero'],
    ['-1', 'La cantidad debe ser mayor que cero'],
  ])('rechaza «%s»', (text, error) => {
    expect(readQuantity(text)).toEqual({ error });
  });
});
