import { aGarment, aProduct, ids, variantId, withoutCost } from '@/test/fixtures';
import type { ResolvedRow } from './resolveRows';
import { changed, targetOf } from './variantTarget';

const garment = aGarment();
const variantOf = (size: string, color: string) => garment.variants.find((v) => v.id === variantId(size, color))!;

// The row of a garment's variant as the API has it.
function rowOf(size: string, color: string, overrides: Partial<ResolvedRow> = {}): ResolvedRow {
  const variant = variantOf(size, color);
  return {
    variantId: variant.id,
    values: variant.axis_values,
    sku: variant.sku,
    barcode: variant.barcode ?? '',
    cost: variant.cost ?? '',
    price: variant.price,
    active: variant.active,
    ...overrides,
  };
}

describe('targetOf', () => {
  it('una fila guarda en su propia variante', () => {
    expect(targetOf(garment, rowOf('M', 'Azul'), false)?.id).toBe(variantId('M', 'Azul'));
  });

  it('una fila nueva guarda en la variante que ya tiene sus valores (de un intento anterior)', () => {
    const row = rowOf('L', 'Rojo', { variantId: undefined });

    expect(targetOf(garment, row, false)?.id).toBe(variantId('L', 'Rojo'));
  });

  it('la fila por defecto de un producto sin ejes guarda en su variante por defecto', () => {
    const row = rowOf('S', 'Rojo', { variantId: undefined });

    expect(targetOf(aProduct(), row, true)?.id).toBe(ids.pipe);
    expect(targetOf(aProduct(), row, false)).toBeUndefined();
  });

  it('una combinación nueva de un producto con ejes no tiene variante todavía', () => {
    const product = { ...garment, variants: garment.variants.slice(0, 2) };

    expect(targetOf(product, rowOf('L', 'Azul', { variantId: undefined }), true)).toBeUndefined();
  });
});

describe('changed', () => {
  it('una fila igual a su variante no se envía, aunque el monto se escriba distinto', () => {
    expect(changed(variantOf('M', 'Rojo'), rowOf('M', 'Rojo', { price: '25000' }))).toBe(false);
  });

  it.each([
    { campo: 'SKU', overrides: { sku: 'CAM-M-ROJA' } },
    { campo: 'código de barras', overrides: { barcode: '' } },
    { campo: 'costo', overrides: { cost: '15500' } },
    { campo: 'precio', overrides: { price: '26000' } },
    { campo: 'estado', overrides: { active: false } },
  ])('un cambio de $campo se envía', ({ overrides }) => {
    expect(changed(variantOf('M', 'Rojo'), rowOf('M', 'Rojo', overrides))).toBe(true);
  });

  it('sin costo visible (quien no edita el catálogo) cuenta como cambio', () => {
    const pipe = withoutCost(aProduct()).variants[0]!;
    const row: ResolvedRow = {
      values: {},
      sku: pipe.sku,
      barcode: '',
      cost: '8333.33',
      price: pipe.price,
      active: true,
    };

    expect(changed(pipe, row)).toBe(true);
  });
});
