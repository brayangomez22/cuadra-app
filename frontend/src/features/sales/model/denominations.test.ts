import { plain } from '@/test/fixtures';
import {
  countedTotal,
  denominationLabel,
  denominations,
  denominationSubtotal,
  parseCount,
  readCounts,
} from './denominations';

describe('parseCount', () => {
  it.each([
    ['', 0],
    ['  ', 0],
    ['3', 3],
    ['007', 7],
    ['10000000', 10_000_000],
  ])('lee «%s» como %d', (text, quantity) => {
    expect(parseCount(text)).toEqual({ ok: true, quantity });
  });

  it.each(['1,5', '1.5', '-1', 'tres', '10000001'])('rechaza «%s»', (text) => {
    expect(parseCount(text)).toEqual({ ok: false });
  });
});

describe('countedTotal', () => {
  it('suma exacto lo contado por denominación', () => {
    expect(
      countedTotal([
        { denomination: '100000', quantity: 3 },
        { denomination: '500', quantity: 7 },
        { denomination: '50', quantity: 1 },
      ]),
    ).toBe('303550');
  });

  it('un conteo vacío es cero', () => {
    expect(countedTotal([])).toBe('0');
  });

  it('no pierde precisión con cantidades grandes', () => {
    expect(countedTotal([{ denomination: '100000', quantity: 10_000_000 }])).toBe('1000000000000');
  });
});

describe('denominations', () => {
  it('son las 11 del peso colombiano, de mayor a menor', () => {
    const values = denominations.map((d) => Number(d.value));
    expect(values).toHaveLength(11);
    expect([...values].sort((a, b) => b - a)).toEqual(values);
  });

  it('nombra billetes, monedas y el 1.000 que es ambos', () => {
    expect(plain(denominationLabel(denominations[0]!))).toBe('Billetes de $ 100.000');
    expect(plain(denominationLabel(denominations.find((d) => d.value === '1000')!))).toBe(
      'Billetes y monedas de $ 1.000',
    );
    expect(plain(denominationLabel(denominations.at(-1)!))).toBe('Monedas de $ 50');
  });
});

describe('readCounts', () => {
  it('ignora las denominaciones vacías o en cero', () => {
    expect(readCounts({ '50000': '2', '1000': '', '500': '0', '50': '  ' })).toEqual({
      counts: [{ denomination: '50000', quantity: 2 }],
      invalid: new Set(),
    });
  });

  it('marca como inválida una cantidad con decimales o letras', () => {
    const read = readCounts({ '100000': '1', '20000': '1,5', '200': 'dos' });
    expect(read.counts).toEqual([{ denomination: '100000', quantity: 1 }]);
    expect(read.invalid).toEqual(new Set(['20000', '200']));
  });
});

describe('denominationSubtotal', () => {
  it('multiplica la denominación por la cantidad', () => {
    expect(denominationSubtotal({ denomination: '20000', quantity: 3 })).toBe('60000');
    expect(denominationSubtotal({ denomination: '50', quantity: 0 })).toBe('0');
  });

  it('acepta la denominación con decimales como la devuelve la API', () => {
    expect(denominationSubtotal({ denomination: '100000.00', quantity: 10_000_000 })).toBe('1000000000000');
  });
});
