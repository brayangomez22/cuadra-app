import { readFactor } from './variantUnitSchema';

describe('readFactor', () => {
  it.each([
    { caso: 'una caja de 12', text: '12', expected: '12' },
    { caso: 'medio metro', text: '0,5', expected: '0.5' },
    { caso: 'vacío', text: '', expected: undefined },
    { caso: 'cero', text: '0', expected: undefined },
    { caso: 'negativo', text: '-3', expected: undefined },
    { caso: 'texto', text: 'doce', expected: undefined },
  ])('$caso', ({ text, expected }) => {
    expect(readFactor(text)).toBe(expected);
  });
});
