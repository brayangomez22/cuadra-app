import { ids, unitOf } from '@/test/fixtures';
import { NO_CATEGORY, type ProductFormValues } from './productSchema';
import { defaultVariantInput, toProductInput } from './toProductInput';

function formValues(overrides: Partial<ProductFormValues> = {}): ProductFormValues {
  return {
    sku: 'CEM-50',
    barcode: '',
    name: 'Cemento gris 50kg',
    description: '',
    category_id: NO_CATEGORY,
    base_unit_id: unitOf('bulto').id,
    tax_rate: '0.19',
    cost: '28000',
    price: '32000',
    attributes: {},
    variants: [],
    ...overrides,
  };
}

const variant = { sku: 'CEM-50', barcode: '', cost: '28000', price: '32000' };

describe('toProductInput', () => {
  it('omite el código de barras y la descripción vacíos, y sin categoría envía null', () => {
    const input = toProductInput(formValues(), variant);

    expect(input).toEqual({
      sku: 'CEM-50',
      name: 'Cemento gris 50kg',
      category_id: null,
      base_unit_id: unitOf('bulto').id,
      cost: '28000',
      price: '32000',
      tax_rate: '0.19',
      attributes: {},
    });
  });

  it('envía la categoría, la descripción y el código de la variante por defecto', () => {
    const values = formValues({ name: 'Camiseta básica', description: 'Algodón', category_id: ids.plumbing });

    const input = toProductInput(values, { sku: 'CAM-S-ROJO', barcode: '7709990001', cost: '15000', price: '25000' });

    expect(input).toMatchObject({
      sku: 'CAM-S-ROJO',
      barcode: '7709990001',
      description: 'Algodón',
      category_id: ids.plumbing,
      price: '25000',
    });
  });

  it('omite los atributos opcionales vacíos y conserva el sí/no falso', () => {
    const values = formValues({ attributes: { marca: 'Acme', material: '', peso: '2.5', importado: false } });

    expect(toProductInput(values, variant).attributes).toEqual({ marca: 'Acme', peso: '2.5', importado: false });
  });
});

describe('defaultVariantInput', () => {
  const garmentRow = {
    values: { talla: 'S', color: 'Rojo' },
    sku: 'CAM-S-ROJO',
    barcode: '',
    cost: '15000',
    price: '25000',
    active: true,
  };

  it('un producto simple (ferretería) envía los datos del formulario', () => {
    expect(defaultVariantInput(formValues({ barcode: '7701234' }))).toEqual({
      sku: 'CEM-50',
      barcode: '7701234',
      cost: '28000',
      price: '32000',
    });
  });

  it('con la matriz (ropa), envía los de la fila de la variante por defecto, también su código vacío', () => {
    const values = formValues({ sku: 'CAM', barcode: '7709990001' });

    expect(defaultVariantInput(values, garmentRow)).toEqual({
      sku: 'CAM-S-ROJO',
      barcode: '',
      cost: '15000',
      price: '25000',
    });
  });

  it('un producto inactivo conserva su precio', () => {
    expect(defaultVariantInput(formValues(), undefined, '30000.00').price).toBe('30000.00');
    expect(defaultVariantInput(formValues(), garmentRow, '24000.00').price).toBe('24000.00');
  });
});
