import { cashResult } from './cashResult';

type Case = {
  name: string;
  closing: Parameters<typeof cashResult>[0];
  expected: ReturnType<typeof cashResult>;
};

const cases: Case[] = [
  {
    name: 'la caja cuadra',
    closing: { result: 'exact', difference: '0.00' },
    expected: { label: 'Cuadra', tone: 'success' },
  },
  {
    name: 'faltan $ 10,50: menos de una moneda grande',
    closing: { result: 'shortage', difference: '-10.50' },
    expected: { label: 'Faltante menor', tone: 'warning' },
  },
  {
    name: 'faltan $ 999',
    closing: { result: 'shortage', difference: '-999.00' },
    expected: { label: 'Faltante menor', tone: 'warning' },
  },
  {
    name: 'faltan $ 1.000: un billete',
    closing: { result: 'shortage', difference: '-1000.00' },
    expected: { label: 'Faltante', tone: 'danger' },
  },
  {
    name: 'sobran $ 50',
    closing: { result: 'overage', difference: '50.00' },
    expected: { label: 'Sobrante menor', tone: 'warning' },
  },
  {
    name: 'sobran $ 20.000',
    closing: { result: 'overage', difference: '20000.00' },
    expected: { label: 'Sobrante', tone: 'warning' },
  },
];

describe('cashResult', () => {
  it.each(cases)('$name', ({ closing, expected }) => {
    expect(cashResult(closing)).toEqual(expected);
  });
});
