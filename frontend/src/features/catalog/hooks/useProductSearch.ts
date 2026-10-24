import { useState } from 'react';
import { useDebouncedValue } from '@/shared/hooks/useDebouncedValue';
import { useProductSearchResults, useSearchProductsNow } from '../api/productSearch';
import { findByCode, sellables, type Sellable } from '../model/variants/variants';

/**
 * The search of ProductPicker: the sellable variants of the active products
 * that match what is typed. Enter searches at once, without waiting for the
 * debounce, and picks the variant with that exact code, or the only option.
 */
export function useProductSearch({ enabled, onPick }: { enabled: boolean; onPick: (item: Sellable) => void }) {
  const searchNow = useSearchProductsNow();
  const [text, setText] = useState('');
  const typed = useDebouncedValue(text.trim());
  const [submitted, setSubmitted] = useState<string | null>(null);
  const term = submitted ?? typed;
  const products = useProductSearchResults(term, { enabled });
  const page = term && !products.isPlaceholderData ? products.data?.items : undefined;

  function change(next: string) {
    setText(next);
    setSubmitted(null);
  }

  function pick(item: Sellable) {
    change('');
    onPick(item);
  }

  async function submit() {
    const q = text.trim();
    if (!q) return;
    setSubmitted(q);
    try {
      const found = await searchNow(q);
      const options = sellables(found);
      const chosen = findByCode(found, q) ?? (options.length === 1 ? options[0] : undefined);
      if (chosen) pick(chosen);
    } catch {
      // The results area shows the failure of the same query.
    }
  }

  return { text, change, term, products, results: page && sellables(page), pick, submit };
}
