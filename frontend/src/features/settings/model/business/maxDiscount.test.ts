import { discountPercentText, readMaxDiscount } from './maxDiscount';

describe('readMaxDiscount', () => {
  it.each([
    { business: 'droguería que no descuenta en el mostrador', text: '0', value: '0' },
    { business: 'almacén de ropa en temporada', text: '15', value: '0.15' },
    { business: 'ferretería', text: '2,5', value: '0.025' },
    { business: 'cualquiera', text: ' 100 ', value: '1' },
    { business: 'cualquiera, vacío', text: '', value: '0' },
  ])('lee el porcentaje como la fracción de la API ($business)', ({ text, value }) => {
    expect(readMaxDiscount(text)).toEqual({ value });
  });

  it.each([
    { text: 'diez', error: 'Escriba un porcentaje, por ejemplo 5 o 2,5' },
    { text: '2,555', error: 'Use máximo 2 decimales' },
    { text: '120', error: 'El descuento va de 0 a 100 %' },
    { text: '-5', error: 'El descuento va de 0 a 100 %' },
  ])('rechaza "$text"', ({ text, error }) => {
    expect(readMaxDiscount(text)).toEqual({ error });
  });
});

describe('discountPercentText', () => {
  it.each([
    { fraction: '0', text: '0' },
    { fraction: '0.15', text: '15' },
    { fraction: '0.025', text: '2,5' },
    { fraction: '1', text: '100' },
  ])('muestra $fraction como $text para el campo', ({ fraction, text }) => {
    expect(discountPercentText(fraction)).toBe(text);
  });
});
