import { aCement, aGarment, aProduct, attributes, axes, unitOf, units, variantId } from '@/test/fixtures';
import { attributeDefault, defaultRow, productDefaults } from './productDefaults';
import { NO_CATEGORY, NO_VALUE } from './productSchema';

const [brand, material, weight, imported] = attributes as [
  (typeof attributes)[0],
  (typeof attributes)[0],
  (typeof attributes)[0],
  (typeof attributes)[0],
];

describe('productDefaults', () => {
  it('un producto nuevo propone la unidad und y el IVA de la empresa', () => {
    const values = productDefaults({ units, definitions: [], defaultTaxRate: '0' });

    expect(values).toMatchObject({ sku: '', name: '', category_id: NO_CATEGORY, tax_rate: '0', variants: [] });
    expect(values.base_unit_id).toBe(unitOf('und').id);
  });

  it.each([
    { negocio: 'ferretería', product: aCement(), cost: '28000', price: '32000' },
    { negocio: 'ferretería', product: aProduct(), cost: '8333,33', price: '10000' },
    { negocio: 'ropa', product: aGarment(), cost: '15000', price: '25000' },
  ])('al editar ($negocio) los montos usan coma decimal', ({ product, cost, price }) => {
    const values = productDefaults({ product, units, definitions: [], defaultTaxRate: '0.19' });

    expect(values).toMatchObject({ cost, price, base_unit_id: product.base_unit_id, variants: [] });
  });

  it('al editar un producto con variantes y el módulo activo, el SKU es la base común y cada variante una fila', () => {
    const values = productDefaults({ product: aGarment(), units, definitions: [], axes, defaultTaxRate: '0.19' });

    expect(values.sku).toBe('CAM');
    expect(values.variants).toHaveLength(6);
    expect(values.variants[0]).toMatchObject({
      variantId: variantId('S', 'Rojo'),
      sku: 'CAM-S-ROJO',
      price: '25000',
    });
  });

  it('sin el módulo de variantes, un producto con variantes edita solo la por defecto', () => {
    const values = productDefaults({ product: aGarment(), units, definitions: [], defaultTaxRate: '0.19' });

    expect(values).toMatchObject({ sku: 'CAM-S-ROJO', variants: [] });
  });
});

describe('attributeDefault', () => {
  it.each([
    { caso: 'texto con valor', definition: brand, value: 'Acme', expected: 'Acme' },
    { caso: 'texto sin valor', definition: brand, value: undefined, expected: '' },
    { caso: 'lista opcional sin valor', definition: material, value: undefined, expected: NO_VALUE },
    {
      caso: 'lista obligatoria sin valor',
      definition: { ...material, required: true },
      value: undefined,
      expected: '',
    },
    { caso: 'número con coma decimal', definition: weight, value: '2.5', expected: '2,5' },
    { caso: 'sí/no sin valor', definition: imported, value: undefined, expected: false },
    { caso: 'sí/no marcado', definition: imported, value: true, expected: true },
  ])('$caso', ({ definition, value, expected }) => {
    expect(attributeDefault(definition, value)).toBe(expected);
  });
});

describe('defaultRow', () => {
  const garment = aGarment();
  const rows = productDefaults({ product: garment, units, definitions: [], axes, defaultTaxRate: '0.19' }).variants;

  it('es la fila de la variante por defecto del producto', () => {
    expect(defaultRow(rows.toReversed(), garment)).toBe(5);
  });

  it('para un producto nuevo o sin variantes en las filas, es la primera', () => {
    expect(defaultRow(rows, undefined)).toBe(0);
    expect(defaultRow(rows, aProduct())).toBe(0);
  });
});
