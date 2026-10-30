import { useEffect, useState } from 'react';
import { useDebouncedValue } from '@/shared/hooks/useDebouncedValue';
import { stringParam, useSearchParamsState } from '@/shared/hooks/useSearchParamsState';

const termParam = stringParam();
// A new search starts at the first page and does not pile up in the history.
const searchOptions = { reset: ['pagina'], replace: true };

/** What the stock is searched by (?buscar=), set once the user stops typing. */
export function useStockSearch() {
  const [term, setTerm] = useSearchParamsState('buscar', termParam, searchOptions);
  const [text, setText] = useState(term);
  const searched = useDebouncedValue(text.trim());
  useEffect(() => {
    if (searched !== term) setTerm(searched);
  }, [searched, term, setTerm]);
  return { term, text, setText };
}
