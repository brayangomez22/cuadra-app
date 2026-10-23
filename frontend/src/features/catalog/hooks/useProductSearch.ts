import { useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { useDebouncedValue } from '@/shared/hooks/useDebouncedValue';
import { catalogKeys } from '../api/keys';
import { fetchProducts, useProducts } from '../api/products';
import type { ProductQuery } from '../api/types';
import { findByCode, sellables, type Sellable } from '../model/variants';

const RESULTS = 8;

const searchQuery = (q: string): ProductQuery => ({ q, status: 'active', limit: RESULTS });

/**
 * The search of ProductPicker: the sellable variants of the active products
 * that match what is typed. Enter searches at once, without waiting for the
 * debounce, and picks the variant with that exact code, or the only option.
 */
export function useProductSearch({ enabled, onPick }: { enabled: boolean; onPick: (item: Sellable) => void }) {
  const queryClient = useQueryClient();
  const [text, setText] = useState('');
  const typed = useDebouncedValue(text.trim());
  const [submitted, setSubmitted] = useState<string | null>(null);
  const term = submitted ?? typed;
  const products = useProducts(searchQuery(term), { enabled: term !== '' && enabled });
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
      const query = searchQuery(q);
      const found = await queryClient.query({
        queryKey: catalogKeys.products(query),
        queryFn: () => fetchProducts(query),
      });
      const options = sellables(found.items);
      const chosen = findByCode(found.items, q) ?? (options.length === 1 ? options[0] : undefined);
      if (chosen) pick(chosen);
    } catch {
      // The results area shows the failure of the same query.
    }
  }

  return { text, change, term, products, results: page && sellables(page), pick, submit };
}
