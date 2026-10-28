import { useState } from 'react';
import { saveVariants } from '../../../api/variants';
import { useStoreProduct } from '../../../api/products';
import type { Product } from '../../../api/types';
import type { ResolvedRow, RowError } from '../../../model/variantMatrix/resolveRows';
import type { ProductFormApi } from './useProductForm';

/** Saves the variant matrix of a product that exists and marks the rows the API rejected. */
export function useSaveMatrix({ form }: { form: ProductFormApi }) {
  const store = useStoreProduct();
  const [failedVariants, setFailedVariants] = useState(0);
  const { setError, setValue } = form;

  function markRows(errors: RowError[]) {
    errors.forEach(({ index, field, message }, i) =>
      setError(`variants.${index}.${field}`, { message }, { shouldFocus: i === 0 }),
    );
  }

  /** The product with all its variants saved, or undefined when some failed (they are marked). */
  async function saveMatrix(saved: Product, rows: ResolvedRow[], defaultIndex: number) {
    const result = await saveVariants({ product: saved, rows, defaultIndex });
    result.variantIds.forEach((id, index) => {
      if (id) setValue(`variants.${index}.variantId`, id);
    });
    if (result.errors.length === 0) return result.product;
    store(result.product);
    setFailedVariants(result.errors.length);
    markRows(result.errors);
    return undefined;
  }

  return { saveMatrix, markRows, failedVariants, clearFailed: () => setFailedVariants(0) };
}
