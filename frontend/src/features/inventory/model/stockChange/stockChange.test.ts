import { describe, expect, it } from 'vitest';
import { adjustmentDelta, balanceAfter, signedQuantity, typedQuantity } from './stockChange';

describe('signedQuantity', () => {
  it.each([
    { name: 'saca unidades rotas de una ferretería', direction: 'out', quantity: '3', expected: '-3' },
    { name: 'saca metros de un rollo', direction: 'out', quantity: '2.5', expected: '-2.5' },
    { name: 'agrega un par de zapatos encontrado', direction: 'in', quantity: '1', expected: '1' },
  ] as const)('$name', ({ direction, quantity, expected }) => {
    expect(signedQuantity(direction, quantity)).toBe(expected);
  });
});

describe('typedQuantity', () => {
  it.each([
    ['lee la coma decimal', '2,5', '2.5'],
    ['lee un entero', '12', '12'],
    ['no acepta cero', '0', null],
    ['no acepta texto', 'dos', null],
    ['no acepta vacío', '', null],
  ])('%s', (_name, text, expected) => {
    expect(typedQuantity(text)).toBe(expected);
  });
});

describe('adjustmentDelta', () => {
  it.each([
    { name: 'sin elegir si sale o entra no hay cambio', direction: '', text: '3', expected: null },
    { name: 'sin cantidad válida no hay cambio', direction: 'out', text: 'x', expected: null },
    { name: 'al sacar es negativo', direction: 'out', text: '0,75', expected: '-0.75' },
    { name: 'al agregar es positivo', direction: 'in', text: '3', expected: '3' },
  ] as const)('$name', ({ direction, text, expected }) => {
    expect(adjustmentDelta(direction, text)).toBe(expected);
  });

  it.each([
    { name: 'ropa: contó 3 camisetas donde había 5', text: '3', current: '5', expected: '-2' },
    { name: 'ferretería: midió 2,5 m de cable donde había 2', text: '2,5', current: '2', expected: '0.5' },
    { name: 'no quedó ninguno', text: '0', current: '4', expected: '-4' },
    { name: 'lo contado es lo que hay', text: '7', current: '7', expected: '0' },
    { name: 'sin saber cuánto hay no hay cambio', text: '3', current: undefined, expected: null },
  ])('al contar lo que hay es la diferencia: $name', ({ text, current, expected }) => {
    expect(adjustmentDelta('count', text, current)).toBe(expected);
  });
});

describe('balanceAfter', () => {
  it.each([
    { name: 'resta unidades', current: '12', delta: '-3', expected: '9' },
    { name: 'suma cantidades fraccionarias sin perder precisión', current: '0.1', delta: '0.2', expected: '0.3' },
    { name: 'resta kilos con cuatro decimales', current: '1.0005', delta: '-0.0005', expected: '1' },
    { name: 'puede quedar negativa', current: '2', delta: '-5', expected: '-3' },
  ])('$name', ({ current, delta, expected }) => {
    expect(balanceAfter(current, delta)).toBe(expected);
  });
});
