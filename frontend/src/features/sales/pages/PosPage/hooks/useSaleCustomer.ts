import { useEffect, useState } from 'react';
import type { ChosenCustomer } from '@/features/customers';
import { loadSaleCustomer, saveSaleCustomer } from '../../../model/saleCustomer';

/**
 * Who the sale in progress goes to, kept in this tab next to the cart under
 * cartKey, and whether the cashier is choosing them (the screen's other keys
 * wait meanwhile).
 */
export function useSaleCustomer(cartKey: string) {
  const [customer, setCustomer] = useState(() => loadSaleCustomer(cartKey));
  const [picking, setPicking] = useState(false);
  useEffect(() => saveSaleCustomer(cartKey, customer), [cartKey, customer]);
  return { customer, setCustomer: (next: ChosenCustomer | null) => setCustomer(next), picking, setPicking };
}

export type SaleCustomer = ReturnType<typeof useSaleCustomer>;
