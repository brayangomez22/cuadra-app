import { ids, unitOf } from '@/test/fixtures';
import type { ResolvedRow } from '../variantMatrix/resolveRows';
import { NO_CATEGORY, type ProductFormValues } from './productSchema';
import { toProductWithVariantsInput } from './toProductWithVariantsInput';

function formValues(overrides: Partial<ProductFormValues>): ProductFormValues {
  return {
    sku: '',
    barcode: '',
    name: '',
    description: '',
    category_id: NO_CATEGORY,
    base_unit_id: unitOf('und').id,
    tax_rate: '0.19',
    cost: '',
    price: '',
    attributes: {},
    variants: [],
    ...overrides,
  };
}

function row(values: Record<string, string>, sku: string, barcode = ''): ResolvedRow {
  return { values, sku, barcode, cost: '15000', price: '30000', active: true };
}

describe('toProductWithVariantsInput', () => {
  it('un almacén de ropa envía todas las variantes, la primera como la por defecto', () => {
    const values = formValues({
      name: 'Camiseta básica',
      category_id: ids.plumbing,
      attributes: { marca: 'Acme', material: '' },
    });
    const rows = [
      row({ talla: 'S', color: 'Rojo' }, 'CAM-S-ROJO'),
      row({ talla: 'M', color: 'Azul' }, 'CAM-M-AZUL', '7709990000004'),
    ];

    expect(toProductWithVariantsInput(values, rows)).toEqual({
      name: 'Camiseta básica',
      category_id: ids.plumbing,
      base_unit_id: unitOf('und').id,
      tax_rate: '0.19',
      attributes: { marca: 'Acme' },
      variants: [
        { sku: 'CAM-S-ROJO', cost: '15000', price: '30000', axis_values: { talla: 'S', color: 'Rojo' } },
        {
          sku: 'CAM-M-AZUL',
          barcode: '7709990000004',
          cost: '15000',
          price: '30000',
          axis_values: { talla: 'M', color: 'Azul' },
        },
      ],
    });
  });

  it('una ferretería con un cable por presentación envía la descripción y sin categoría null', () => {
    const values = formValues({
      name: 'Cable eléctrico N.º 12',
      description: 'Cobre, 600 V',
      base_unit_id: unitOf('m').id,
    });

    const input = toProductWithVariantsInput(values, [row({ presentación: 'Rollo 100 m' }, 'CAB-12-R100')]);

    expect(input).toMatchObject({ description: 'Cobre, 600 V', category_id: null, base_unit_id: unitOf('m').id });
    expect(input.variants).toEqual([
      { sku: 'CAB-12-R100', cost: '15000', price: '30000', axis_values: { presentación: 'Rollo 100 m' } },
    ]);
  });
});
