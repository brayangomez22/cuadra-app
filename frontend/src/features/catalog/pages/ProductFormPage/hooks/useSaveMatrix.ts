import { useState } from 'react';
import { saveVariants } from '../../../api/variants';
import { useStoreProduct } from '../../../api/products';
import type { Product } from '../../../api/types';
import type { ResolvedRow, RowError } from '../../../model/variantMatrix/resolveRows';
import type { ProductFormApi } from './useProductForm';

type Input = {
  form: ProductFormApi;
  /** A new product whose variants failed in part exists already: saving again edits it. */
  onPartial: (product: Product) => void;
};

/** Saves the variant matrix after the product and marks the rows the API rejected. */
export function useSaveMatrix({ form, onPartial }: Input) {
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
    onPartial(result.product);
    setFailedVariants(result.errors.length);
    markRows(result.errors);
    return undefined;
  }

  return { saveMatrix, markRows, failedVariants, clearFailed: () => setFailedVariants(0) };
}
