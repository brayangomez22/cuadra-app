import { resolveRows, type RowDefaults } from './resolveRows';
import type { VariantRow } from './productSchema';

const row = (values: Record<string, string>, overrides: Partial<VariantRow> = {}): VariantRow => ({
  values,
  sku: '',
  barcode: '',
  cost: '',
  price: '',
  active: true,
  ...overrides,
});

describe('resolveRows', () => {
  it.each<{ negocio: string; defaults: RowDefaults; values: Record<string, string>; sku: string }>([
    {
      negocio: 'ropa',
      defaults: { sku: 'cam', cost: '15000', price: '25000' },
      values: { talla: 'M', color: 'Azul' },
      sku: 'CAM-M-AZUL',
    },
    {
      negocio: 'ferretería',
      defaults: { sku: 'tub', cost: '8333,33', price: '10000' },
      values: { medida: '1/2 pulgada' },
      sku: 'TUB-1/2PULGADA',
    },
  ])(
    'una fila vacía ($negocio) toma el SKU generado, el costo y el precio del producto',
    ({ defaults, values, sku }) => {
      const { rows, errors } = resolveRows([row(values)], defaults);

      expect(errors).toEqual([]);
      expect(rows[0]).toMatchObject({ sku, barcode: '', active: true, values });
      expect(rows[0]?.price).toBe(defaults.price.replace(',', '.'));
    },
  );

  it('una fila con datos propios los conserva, con el SKU en mayúsculas', () => {
    const typed = row({ talla: 'L' }, { sku: ' cam-l ', barcode: ' 770999 ', cost: '16000', price: '27000,50' });

    const { rows } = resolveRows([typed], { sku: 'cam', cost: '15000', price: '25000' });

    expect(rows[0]).toMatchObject({ sku: 'CAM-L', barcode: '770999', cost: '16000', price: '27000.50' });
  });

  it('marca el SKU vacío sin base y el de más de 64 caracteres', () => {
    const { errors } = resolveRows([row({ talla: 'S' }), row({ talla: 'M' }, { sku: 'X'.repeat(65) })], {
      sku: '',
      cost: '0',
      price: '1',
    });

    expect(errors).toEqual([
      { index: 0, field: 'sku', message: 'Escriba el SKU' },
      { index: 1, field: 'sku', message: 'El SKU puede tener máximo 64 caracteres' },
    ]);
  });

  it('marca el código de barras de más de 64 caracteres', () => {
    const { errors } = resolveRows([row({ talla: 'S' }, { barcode: '7'.repeat(65) })], {
      sku: 'cam',
      cost: '0',
      price: '1',
    });

    expect(errors).toEqual([
      { index: 0, field: 'barcode', message: 'El código de barras puede tener máximo 64 caracteres' },
    ]);
  });

  it('un costo o precio propio inválido se marca en la fila; uno heredado inválido no', () => {
    const rows = [row({ talla: 'S' }, { cost: '-1', price: '0' }), row({ talla: 'M' })];

    const { errors } = resolveRows(rows, { sku: 'cam', cost: '-5', price: '' });

    expect(errors).toEqual([
      { index: 0, field: 'cost', message: 'El costo no puede ser negativo' },
      { index: 0, field: 'price', message: 'El precio debe ser mayor que cero' },
    ]);
  });
});
