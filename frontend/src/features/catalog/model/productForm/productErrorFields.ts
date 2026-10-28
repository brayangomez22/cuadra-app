import type { Path } from 'react-hook-form';
import { ApiError } from '@/shared/api/errors';
import type { FieldsByErrorCode } from '@/shared/forms/useApiFormErrors';
import type { AttributeDefinition } from '../../api/types';
import type { ProductFormInput } from './productSchema';

type VariantField = 'sku' | 'barcode' | 'cost' | 'price';

// Error codes of POST and PUT /products (api/openapi.yaml) that belong to a
// field of the default variant.
const variantFieldByCode: Record<string, VariantField> = {
  invalid_sku: 'sku',
  sku_taken: 'sku',
  invalid_barcode: 'barcode',
  barcode_taken: 'barcode',
  invalid_cost: 'cost',
  invalid_price: 'price',
  product_inactive: 'price',
};

const productFieldByCode: Record<string, Path<ProductFormInput>> = {
  invalid_product_name: 'name',
  invalid_description: 'description',
  category_not_found: 'category_id',
  unit_not_found: 'base_unit_id',
  unit_required: 'base_unit_id',
  invalid_tax_rate: 'tax_rate',
};

type Options = {
  matrix: boolean;
  defaultIndex: number;
  /**
   * The product and its variants go in one request (POST /products-with-variants):
   * the API does not say which variant an error is of, so their codes go to
   * the form's message; a SKU or barcode in use names its code, and
   * useSaveProduct marks its row (model/variantMatrix/takenCodeRow).
   */
  together?: boolean;
};

/**
 * The field of each error code of saving a product. With the variant matrix,
 * the default variant's fields are those of its row.
 */
export function productErrorFields({ matrix, defaultIndex, together = false }: Options) {
  const variantFields = together
    ? []
    : Object.entries(variantFieldByCode).map(([code, field]) => [
        code,
        matrix ? (`variants.${String(defaultIndex)}.${field}` as const) : field,
      ]);
  return { ...productFieldByCode, ...Object.fromEntries(variantFields) } as FieldsByErrorCode<ProductFormInput>;
}

// Errors of one attribute name it in details.attribute.
const attributeErrorCodes = new Set(['unknown_attribute', 'invalid_attribute_value', 'attribute_required']);

/** The attribute field an error of the API names, when the form has it. */
export function attributeErrorField(error: unknown, definitions: AttributeDefinition[]) {
  if (!(error instanceof ApiError) || !attributeErrorCodes.has(error.code ?? '')) return undefined;
  const { attribute } = error.details;
  return definitions.some((d) => d.key === attribute) ? (`attributes.${attribute ?? ''}` as const) : undefined;
}
