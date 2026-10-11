import { plain } from '@/test/fixtures';
import { formatDateTime, formatMoney, formatPercent, formatQuantity, parseDecimalInput, parseMoneyInput } from './format';

describe('formatMoney', () => {
  it.each([
    ['12500.00', '$ 12.500'],
    ['12500', '$ 12.500'],
    ['12500.5', '$ 12.500,50'],
    ['8333.3333', '$ 8.333,33'],
    ['0.005', '$ 0,01'],
    ['99999999999999.99', '$ 99.999.999.999.999,99'],
  ])('formatea %s como %s, sin pasar por float', (value, expected) => {
    expect(plain(formatMoney(value))).toBe(expected);
  });
});

describe('formatQuantity', () => {
  it.each([
    ['2.5', '2,5'],
    ['1250', '1.250'],
    ['-3', '-3'],
    ['0.1234', '0,1234'],
    ['12345678901234.1234', '12.345.678.901.234,1234'],
  ])('formatea %s como %s', (value, expected) => {
    expect(formatQuantity(value)).toBe(expected);
  });

  it('agrega el signo + a una entrada cuando se pide', () => {
    expect(formatQuantity('10', { signed: true })).toBe('+10');
    expect(formatQuantity('-3', { signed: true })).toBe('-3');
  });
});

describe('formatPercent', () => {
  it.each([
    ['0.19', '19 %'],
    ['0.05', '5 %'],
    ['0', '0 %'],
    ['0.085', '8,5 %'],
  ])('formatea la tasa %s como %s', (value, expected) => {
    expect(plain(formatPercent(value, 'rate'))).toBe(expected);
  });

  it.each([
    ['16.6667', '16,67 %'],
    ['-5', '-5 %'],
  ])('formatea el porcentaje %s como %s', (value, expected) => {
    expect(plain(formatPercent(value, 'percent'))).toBe(expected);
  });
});

describe('formatDateTime', () => {
  it('muestra la fecha y la hora de Bogotá en formato dd/mm/aaaa', () => {
    expect(plain(formatDateTime('2026-10-10T20:05:00Z'))).toBe('10/10/2026, 3:05 p. m.');
    expect(plain(formatDateTime('2026-10-11T03:30:00Z'))).toBe('10/10/2026, 10:30 p. m.');
  });
});

describe('parseDecimalInput', () => {
  it.each([
    ['12500', '12500'],
    ['12500,50', '12500.50'],
    ['12500.50', '12500.50'],
    ['2,5', '2.5'],
    ['2.5', '2.5'],
    [' 0,1234 ', '0.1234'],
    ['0', '0'],
    ['-3', '-3'],
    ['1.5000', '1.5000'],
  ])('acepta %j como %j', (text, value) => {
    expect(parseDecimalInput(text)).toEqual({ ok: true, value });
  });

  it.each(['12.500', '1.250.000', '-2.500'])('rechaza %j por ambiguo: parece un punto de miles', (text) => {
    expect(parseDecimalInput(text)).toEqual({ ok: false, reason: 'thousands' });
  });

  it.each(['', 'abc', '12,5,0', '1 000', '$12500', '12e3', '.5', '5.'])('rechaza %j porque no es un número', (text) => {
    expect(parseDecimalInput(text)).toEqual({ ok: false, reason: 'invalid' });
  });

  it('rechaza más de 4 decimales', () => {
    expect(parseDecimalInput('1,12345')).toEqual({ ok: false, reason: 'decimals' });
  });

  it('rechaza más de 14 dígitos enteros', () => {
    expect(parseDecimalInput('123456789012345')).toEqual({ ok: false, reason: 'too_large' });
  });
});

describe('parseMoneyInput', () => {
  it.each([
    ['100.000', '100000'],
    ['1.250.000', '1250000'],
    ['1.250.000,50', '1250000.50'],
    ['50000', '50000'],
    ['12500,5', '12500.5'],
  ])('acepta %j como lo escribe un cajero: %j', (text, value) => {
    expect(parseMoneyInput(text)).toEqual({ ok: true, value });
  });

  it.each([
    ['1.25.000', 'invalid'],
    ['100.000,123', 'invalid'],
    ['abc', 'invalid'],
  ] as const)('rechaza %j', (text, reason) => {
    expect(parseMoneyInput(text)).toEqual({ ok: false, reason });
  });
});
