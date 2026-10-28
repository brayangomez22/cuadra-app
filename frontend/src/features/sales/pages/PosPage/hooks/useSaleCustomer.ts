import { useEffect, useState } from 'react';
import type { ChosenCustomer } from '@/features/customers';
import { loadSaleCustomer, saveSaleCustomer } from '../../../model/saleCustomer';

/** Who the sale in progress goes to, kept in this tab next to the cart under cartKey. */
export function useSaleCustomer(cartKey: string) {
  const [customer, setCustomer] = useState(() => loadSaleCustomer(cartKey));
  useEffect(() => saveSaleCustomer(cartKey, customer), [cartKey, customer]);
  return { customer, setCustomer: (next: ChosenCustomer | null) => setCustomer(next) };
}

export type SaleCustomer = ReturnType<typeof useSaleCustomer>;
