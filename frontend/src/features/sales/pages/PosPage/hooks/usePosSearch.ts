import { useState } from 'react';
import { sellables, useProductSearchResults, useSearchProductsNow, type Sellable } from '@/features/catalog';
import { useDebouncedValue } from '@/shared/hooks/useDebouncedValue';
import { readSearch, searchOutcome, type SearchOutcome } from '../../../model/searchText';

/** Adds the variant to the sale; quantity is a decimal of the API ("1", "2.5"). */
export type OnAdd = (sellable: Sellable, quantity: string) => void;

/**
 * The state of the POS search: the text typed, the variants found and the
 * one the arrows chose. Enter searches at once, without the debounce.
 */
export function usePosSearch(onAdd: OnAdd) {
  const search = useSearchProductsNow();
  const [text, setText] = useState('');
  const [active, setActive] = useState(-1);
  const [submitted, setSubmitted] = useState<string | null>(null);
  const [notFound, setNotFound] = useState<string | null>(null);
  const { quantity, term: typed } = readSearch(text);
  const debounced = useDebouncedValue(typed);
  const term = submitted ?? debounced;
  const products = useProductSearchResults(term);
  const page = term && !products.isPlaceholderData ? products.data?.items : undefined;
  const results = page && sellables(page);

  function edit(value: string) {
    setText(value);
    setSubmitted(null);
    setActive(-1);
    setNotFound(null);
  }

  function add(sellable: Sellable, amount = quantity) {
    onAdd(sellable, amount);
    edit('');
  }

  /** Searches value (the text typed by default) at once and says what came of it. */
  async function searchNow(value = text): Promise<SearchOutcome['type'] | undefined> {
    const read = readSearch(value);
    if (!read.term) return undefined;
    setText(value);
    setSubmitted(read.term);
    try {
      const outcome = searchOutcome(await search(read.term), read.term);
      if (outcome.type === 'add') {
        add(outcome.sellable, read.quantity);
      } else if (outcome.type === 'not_found') {
        // Clear the text so the next scan does not append to this code.
        edit('');
        setNotFound(read.term);
      } else {
        setActive(0);
      }
      return outcome.type;
    } catch {
      // The results area shows the failure of the same query.
      return undefined;
    }
  }

  /** A code read by the camera, with the quantity typed before it ("3*"), if any. */
  const scan = (code: string) => searchNow(typed ? code : text + code);

  return { text, term, products, results, notFound, active, setActive, edit, add, searchNow, scan };
}

export type PosSearch = ReturnType<typeof usePosSearch>;
