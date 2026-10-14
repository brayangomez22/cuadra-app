import { plain } from '@/test/fixtures';
import { countedTotal, denominationLabel, denominations, moneyField, parseCount } from './cash';

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

describe('moneyField', () => {
  const field = moneyField('positive', 'Escriba el valor');

  it.each([
    ['150.000', '150000'],
    ['12500,5', '12500.5'],
    ['1.250.000,50', '1250000.50'],
  ])('lee «%s» como %s', (text, value) => {
    expect(field.parse(text)).toBe(value);
  });

  it.each([
    ['', 'Escriba el valor'],
    ['0', 'El valor debe ser mayor que cero'],
    ['10,555', 'Escriba un valor, por ejemplo 50.000 o 50000'],
    ['cien', 'Escriba un valor, por ejemplo 50.000 o 50000'],
  ])('rechaza «%s»', (text, message) => {
    const result = field.safeParse(text);
    expect(result.success).toBe(false);
    expect(result.error?.issues[0]?.message).toBe(message);
  });

  it('con mínimo cero acepta 0 y rechaza negativos', () => {
    const float = moneyField('nonnegative', 'Escriba la base');
    expect(float.parse('0')).toBe('0');
    expect(float.safeParse('-5').error?.issues[0]?.message).toBe('El valor no puede ser negativo');
  });
});
