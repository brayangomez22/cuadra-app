import { moneyField } from './moneyField';

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
