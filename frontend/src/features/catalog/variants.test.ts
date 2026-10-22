import { aCement, aGarment, aProduct, boxOf12, ids, packOf3, variantBarcode, variantId, withUnits } from '@/test/fixtures';
import { defaultVariant, findByCode, hasAxes, sellables, suggestedUnitPrice, variantName } from './variants';

describe('variantes', () => {
  it('la variante por defecto es la creada con el producto, aunque su id no sea el del producto', () => {
    expect(defaultVariant(aProduct()).id).toBe(ids.pipe);
    expect(defaultVariant(aGarment()).id).toBe(variantId('S', 'Rojo'));
    expect(defaultVariant(aGarment()).id).not.toBe(ids.garment);
  });

  it('nombra la variante como la tirilla: el producto y, si tiene ejes, su etiqueta', () => {
    expect(variantName('Tubo PVC', '')).toBe('Tubo PVC');
    expect(variantName('Camiseta básica', 'Rojo · M')).toBe('Camiseta básica (Rojo · M)');
  });

  it('sabe si un producto varía en ejes', () => {
    expect(hasAxes(aProduct())).toBe(false);
    expect(hasAxes(aGarment())).toBe(true);
  });

  it('lista las variantes vendibles: activas y de productos activos', () => {
    const garment = aGarment();
    garment.variants[1]!.active = false;
    const options = sellables([aProduct(), garment, aCement({ active: false })]);
    expect(options.map((o) => o.variant.id)).toEqual([
      ids.pipe,
      variantId('S', 'Rojo'),
      variantId('M', 'Rojo'),
      variantId('M', 'Azul'),
      variantId('L', 'Rojo'),
      variantId('L', 'Azul'),
    ]);
  });

  it('encuentra la variante por su código de barras o su SKU exactos', () => {
    const products = [aCement({ barcode: '7701234567890' }), aGarment()];
    const tests: [string, string | undefined][] = [
      ['7701234567890', ids.cement],
      ['cem-50', ids.cement],
      [variantBarcode('M', 'Azul'), variantId('M', 'Azul')],
      ['cam-l-rojo', variantId('L', 'Rojo')],
      ['CAM', undefined],
    ];
    for (const [code, want] of tests) {
      expect(findByCode(products, code)?.variant.id, code).toBe(want);
    }
  });

  it('encuentra la unidad alternativa por su código de barras: la caja de tubos', () => {
    const tubes = withUnits(aProduct(), [boxOf12]);
    const found = findByCode([tubes], boxOf12.barcode!);
    expect(found?.variant.id).toBe(ids.pipe);
    expect(found?.unit?.unit.code).toBe('caja');
    expect(findByCode([tubes], 'TUB-PVC-12')?.unit).toBeUndefined();
  });

  it('encuentra el paquete de una variante de ropa por su código', () => {
    const red = variantId('M', 'Rojo');
    const found = findByCode([withUnits(aGarment(), [packOf3], red)], packOf3.barcode!);
    expect(found?.variant.id).toBe(red);
    expect(found?.unit?.factor).toBe('3');
  });

  it.each([
    ['10000.00', '12', '120000.00'],
    ['2000.00', '100', '200000.00'],
    ['8403.3613', '3', '25210.08'],
    ['1234.5678', '0.5', '617.28'],
    ['0.01', '0.5', '0.01'],
  ])('sugiere el precio proporcional de %s × %s redondeado a centavos: %s', (price, factor, expected) => {
    expect(suggestedUnitPrice(price, factor)).toBe(expected);
  });
});
