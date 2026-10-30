import { roundToWhole, fromUnits, fromUnitsTrimmed, mulRound2, mulTrunc2, toUnits } from './decimal';

describe('decimal', () => {
  it('lee y escribe decimales de la API sin pasar por float', () => {
    expect(toUnits('12500.5')).toBe(125005000n);
    expect(toUnits('0.0001')).toBe(1n);
    expect(toUnits('-3')).toBe(-30000n);
    expect(fromUnits(toUnits('12500.5'))).toBe('12500.50');
    expect(fromUnits(toUnits('8403.3613'))).toBe('8403.3613');
    expect(fromUnits(toUnits('0'))).toBe('0.00');
    expect(fromUnits(toUnits('-1.5'))).toBe('-1.50');
  });

  it('escribe cantidades sin ceros a la derecha', () => {
    expect(fromUnitsTrimmed(toUnits('3.50'))).toBe('3.5');
    expect(fromUnitsTrimmed(toUnits('2.0000'))).toBe('2');
    expect(fromUnitsTrimmed(toUnits('100'))).toBe('100');
  });

  it('rechaza lo que no es un decimal de hasta 4 cifras', () => {
    expect(() => toUnits('1.23456')).toThrow();
    expect(() => toUnits('abc')).toThrow();
    expect(() => toUnits('1,5')).toThrow();
  });

  it('multiplica cantidades fraccionarias sin perder precisión', () => {
    // 0.1 × 3 is 0.30000000000000004 in float.
    expect(fromUnits(mulRound2(toUnits('0.1'), toUnits('3')))).toBe('0.30');
    expect(fromUnits(mulRound2(toUnits('2.5'), toUnits('8403.3613')))).toBe('21008.40');
  });

  it.each([
    ['1.005', '1', '1.01'],
    ['1.0049', '1', '1.00'],
    ['20908.40', '0.19', '3972.60'],
    ['-1.005', '1', '-1.01'],
  ])('redondea half-up a 2 decimales: %s × %s = %s', (a, b, want) => {
    expect(fromUnits(mulRound2(toUnits(a), toUnits(b)))).toBe(want);
  });

  it('trunca a centavos', () => {
    expect(fromUnits(mulTrunc2(toUnits('10000'), toUnits('0.3333')))).toBe('3333.00');
    expect(fromUnits(mulTrunc2(toUnits('99.99'), toUnits('0.15')))).toBe('14.99');
  });
});

describe('roundToWhole', () => {
  it.each([
    ['12500.5', '12501.00'],
    ['12500.49', '12500.00'],
    ['12500', '12500.00'],
    ['-12500.5', '-12501.00'],
  ])('%s queda en %s, como la muestra formatMoney', (value, expected) => {
    expect(fromUnits(roundToWhole(toUnits(value)))).toBe(expected);
  });
});
