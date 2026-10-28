import { ApiError } from '@/shared/api/errors';
import { attributes } from '@/test/fixtures';
import { attributeErrorField, productErrorFields } from './productErrorFields';

const apiError = (code: string, details: Record<string, string> = {}) =>
  new ApiError(422, { error: { code, message: 'Mensaje de la API', details } });

describe('productErrorFields', () => {
  it.each([
    { code: 'sku_taken', field: 'sku' },
    { code: 'barcode_taken', field: 'barcode' },
    { code: 'invalid_product_name', field: 'name' },
    { code: 'category_not_found', field: 'category_id' },
    { code: 'unit_required', field: 'base_unit_id' },
    { code: 'product_inactive', field: 'price' },
    { code: 'invalid_tax_rate', field: 'tax_rate' },
  ])('sin matriz, $code va a $field', ({ code, field }) => {
    expect(productErrorFields({ matrix: false, defaultIndex: 0 })[code]).toBe(field);
  });

  it('con matriz, los campos de variante van a la fila por defecto y los del producto a su campo', () => {
    const fields = productErrorFields({ matrix: true, defaultIndex: 2 });

    expect(fields.sku_taken).toBe('variants.2.sku');
    expect(fields.invalid_cost).toBe('variants.2.cost');
    expect(fields.invalid_product_name).toBe('name');
  });

  it('con el producto y sus variantes juntos, los códigos de variante van al mensaje del formulario', () => {
    const fields = productErrorFields({ matrix: true, defaultIndex: 0, together: true });

    expect(fields.sku_taken).toBeUndefined();
    expect(fields.barcode_taken).toBeUndefined();
    expect(fields.invalid_product_name).toBe('name');
  });

  it('un código que no es de un campo no tiene destino', () => {
    expect(productErrorFields({ matrix: false, defaultIndex: 0 }).forbidden).toBeUndefined();
  });
});

describe('attributeErrorField', () => {
  it('un error de atributo va al atributo que nombra details.attribute', () => {
    expect(attributeErrorField(apiError('attribute_required', { attribute: 'marca' }), attributes)).toBe(
      'attributes.marca',
    );
    expect(attributeErrorField(apiError('invalid_attribute_value', { attribute: 'peso' }), attributes)).toBe(
      'attributes.peso',
    );
  });

  it.each([
    { caso: 'un atributo que la empresa no define', error: apiError('unknown_attribute', { attribute: 'talla' }) },
    { caso: 'otro código con details.attribute', error: apiError('sku_taken', { attribute: 'marca' }) },
    { caso: 'un error que no es de la API', error: new Error('sin conexión') },
  ])('$caso no va a un atributo', ({ error }) => {
    expect(attributeErrorField(error, attributes)).toBeUndefined();
  });
});
