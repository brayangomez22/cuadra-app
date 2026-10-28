import { useState } from 'react';
import { useDebouncedValue } from '@/shared/hooks/useDebouncedValue';
import { useCustomers } from '../api/customers';
import type { Customer } from '../api/types';

// A quick search shows the first few: the exact document comes first.
const LIMIT = 8;

/**
 * The POS's customer search: the active customers whose name or document
 * matches what was typed, once the cashier stops typing, and the option
 * the arrows move to.
 */
export function useCustomerSearch() {
  const [text, setText] = useState('');
  const [active, setActive] = useState(0);
  const term = useDebouncedValue(text.trim());
  const customers = useCustomers({ q: term, status: 'active', limit: LIMIT }, { enabled: term !== '' });
  const results: Customer[] | undefined = term && !customers.isPlaceholderData ? customers.data?.items : undefined;

  function move(step: 1 | -1) {
    const count = results?.length ?? 0;
    if (count > 0) setActive((current) => (current + step + count) % count);
  }

  return {
    text,
    change: (next: string) => {
      setText(next);
      setActive(0);
    },
    term,
    results,
    failed: customers.isError,
    active,
    move,
  };
}

export type CustomerSearch = ReturnType<typeof useCustomerSearch>;
