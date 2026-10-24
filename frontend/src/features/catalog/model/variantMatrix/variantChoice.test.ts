import { axes } from '@/test/fixtures';
import type { VariantRow } from '../productForm/productSchema';
import { MAX_COMBINATIONS, chosenOf, nextRows, toggleValue, withAxesOf } from './variantChoice';

const row = (values: Record<string, string>, overrides: Partial<VariantRow> = {}): VariantRow => ({
  values,
  sku: '',
  barcode: '',
  cost: '',
  price: '',
  active: true,
  ...overrides,
});

describe('chosenOf', () => {
  it('reúne los valores que usan las filas, por eje y sin repetir', () => {
    const rows = [row({ talla: 'S', color: 'Rojo' }), row({ talla: 'M', color: 'Rojo' })];

    expect(chosenOf(rows)).toEqual({ talla: ['S', 'M'], color: ['Rojo'] });
  });
});

describe('withAxesOf', () => {
  it('agrega a los ejes de la empresa los valores y ejes que usan sus variantes y ya no define', () => {
    const rows = [row({ talla: 'XL', presentación: 'Galón' }, { variantId: 'v1' })];

    const all = withAxesOf(axes, rows);

    expect(all.map((axis) => [axis.name, axis.values])).toEqual([
      ['talla', ['S', 'M', 'L', 'XL']],
      ['color', ['Rojo', 'Azul']],
      ['presentación', ['Galón']],
    ]);
  });
});

describe('toggleValue', () => {
  it('agrega un valor que no estaba y quita uno que estaba', () => {
    expect(toggleValue({ talla: ['S'] }, 'talla', 'M')).toEqual({ talla: ['S', 'M'] });
    expect(toggleValue({ talla: ['S', 'M'] }, 'talla', 'S')).toEqual({ talla: ['M'] });
    expect(toggleValue({}, 'color', 'Rojo')).toEqual({ color: ['Rojo'] });
  });
});

describe('nextRows', () => {
  it('cada combinación es una fila nueva y las existentes se conservan con lo escrito', () => {
    const existing = row({ talla: 'S', color: 'Rojo' }, { variantId: 'v1', sku: 'CAM-S-ROJO' });

    const rows = nextRows(axes, { talla: ['S', 'M'], color: ['Rojo'] }, [existing]);

    expect(rows).toEqual([existing, row({ talla: 'M', color: 'Rojo' })]);
  });

  it('una presentación sola (un eje) da una fila por valor', () => {
    const presentations = [{ id: 'p', name: 'presentación', values: ['Galón', 'Cuarto'] }];

    expect(nextRows(presentations, { presentación: ['Cuarto', 'Galón'] }, [])).toEqual([
      row({ presentación: 'Galón' }),
      row({ presentación: 'Cuarto' }),
    ]);
  });

  it(`rechaza pasar de ${String(MAX_COMBINATIONS)} combinaciones`, () => {
    const many = (prefix: string) => Array.from({ length: 11 }, (_, i) => `${prefix}${String(i)}`);
    const wide = [
      { id: 'a', name: 'a', values: many('a') },
      { id: 'b', name: 'b', values: many('b') },
    ];

    expect(nextRows(wide, { a: many('a'), b: many('b') }, [])).toBeNull();
  });
});
