import { useMemo } from 'react';
import { useNavigate } from 'react-router';
import { paths } from '@/app/routes/paths';
import { describeError } from '@/shared/api/errors';
import { useApiFormErrors } from '@/shared/forms/useApiFormErrors';
import { useToast } from '@/shared/ui';
import { createProductWithVariants, saveProduct, useStoreProduct } from '../../../api/products';
import type { AttributeDefinition, Product } from '../../../api/types';
import { defaultRow } from '../../../model/productForm/productDefaults';
import { attributeErrorField, productErrorFields } from '../../../model/productForm/productErrorFields';
import type { ProductFormValues, VariantRow } from '../../../model/productForm/productSchema';
import { resolveRows, type ResolvedRow } from '../../../model/variantMatrix/resolveRows';
import { defaultVariantInput, toProductInput } from '../../../model/productForm/toProductInput';
import { toProductWithVariantsInput } from '../../../model/productForm/toProductWithVariantsInput';
import { defaultVariant } from '../../../model/variants/variants';
import type { ProductFormApi } from './useProductForm';
import { useSaveMatrix } from './useSaveMatrix';

type Input = {
  form: ProductFormApi;
  /** The saved product, if any: saving updates it. */
  current?: Product;
  definitions: AttributeDefinition[];
  /** The rows of the variant matrix as typed; empty without it. */
  rows: VariantRow[];
};

/**
 * Saves the form. A new product with the variant matrix goes with all its
 * rows in one request; otherwise, the product and then the rows that changed.
 * When it is all saved, it goes back to the list; when the API rejects
 * something, it marks it on its field and stays.
 */
export function useSaveProduct({ form, current, definitions, rows }: Input) {
  const navigate = useNavigate();
  const store = useStoreProduct();
  const { toast } = useToast();
  const matrix = rows.length > 0;
  const together = matrix && !current;
  const defaultIndex = defaultRow(rows, current);
  const fields = useMemo(
    () => productErrorFields({ matrix, defaultIndex, together }),
    [matrix, defaultIndex, together],
  );
  const { formError, handleError, clearFormError } = useApiFormErrors(form.setError, fields);
  const { saveMatrix, markRows, failedVariants, clearFailed } = useSaveMatrix({ form });

  function showError(error: unknown) {
    const attribute = attributeErrorField(error, definitions);
    if (attribute) form.setError(attribute, { message: describeError(error).message }, { shouldFocus: true });
    else handleError(error);
  }

  // saveAll saves the product and its matrix: the product, or undefined when something failed (it is marked).
  async function saveAll(values: ProductFormValues, rows: ResolvedRow[] | undefined) {
    const keptPrice = current?.active === false ? defaultVariant(current).price : undefined;
    const body = toProductInput(values, defaultVariantInput(values, rows?.[defaultIndex], keptPrice));
    try {
      if (rows && together) return await createProductWithVariants(toProductWithVariantsInput(values, rows));
      const saved = await saveProduct(current, body);
      return rows ? await saveMatrix(saved, rows, defaultIndex) : saved;
    } catch (error) {
      showError(error);
      return undefined;
    }
  }

  async function save(values: ProductFormValues) {
    clearFormError();
    clearFailed();
    const resolved = matrix ? resolveRows(values.variants, values) : undefined;
    if (resolved && resolved.errors.length > 0) {
      markRows(resolved.errors);
      return;
    }
    const saved = await saveAll(values, resolved?.rows);
    if (!saved) return;
    store(saved);
    toast({ tone: 'success', title: 'Producto guardado', description: saved.name });
    void navigate(paths.products.list());
  }

  return { submit: form.handleSubmit(save), formError, failedVariants };
}
