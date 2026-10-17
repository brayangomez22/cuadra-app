import { aGarment, axes } from '@/test/fixtures';
import { axisLabel, combinations, generatedSku, skuPrefix } from './variantMatrix';

describe('matriz de variantes', () => {
  it('la etiqueta ordena los valores por nombre de eje, como la API', () => {
    expect(axisLabel({ talla: 'M', color: 'Rojo' })).toBe('Rojo · M');
    expect(axisLabel({ presentacion: 'Caja x 10' })).toBe('Caja x 10');
    expect(axisLabel({})).toBe('');
  });

  it('combina los valores elegidos en el orden de los ejes y de sus valores', () => {
    expect(combinations(axes, { color: ['Azul', 'Rojo'], talla: ['S', 'L'] })).toEqual([
      { talla: 'S', color: 'Rojo' },
      { talla: 'S', color: 'Azul' },
      { talla: 'L', color: 'Rojo' },
      { talla: 'L', color: 'Azul' },
    ]);
  });

  it('un eje sin valores elegidos no cuenta, y sin valores no hay combinaciones', () => {
    expect(combinations(axes, { talla: ['M'], color: [] })).toEqual([{ talla: 'M' }]);
    expect(combinations(axes, {})).toEqual([]);
  });

  it('arma el SKU con la base y los valores en el orden de los ejes, en mayúsculas y sin espacios ni tildes', () => {
    const tests: [string, Record<string, string>, string][] = [
      ['cam', { talla: 'M', color: 'Azul' }, 'CAM-M-AZUL'],
      ['TOR', { medida: '1/4 pulgada' }, 'TOR-1/4PULGADA'],
      ['beb', { sabor: 'Limón' }, 'BEB-LIMON'],
      ['', { talla: 'M' }, ''],
    ];
    for (const [base, values, want] of tests) expect(generatedSku(base, values)).toBe(want);
  });

  it('la base del SKU de un producto con variantes es lo que comparten sus SKU', () => {
    expect(skuPrefix(aGarment())).toBe('CAM');
  });
});
