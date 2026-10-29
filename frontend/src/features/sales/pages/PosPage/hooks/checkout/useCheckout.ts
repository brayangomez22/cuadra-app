import { useState } from 'react';
import type { ChosenCustomer } from '@/features/customers';
import { useRefreshCurrentCash } from '../../../../api/cashSessions';
import type { Sale } from '../../../../api/types';
import type { CartState } from '../useCart';
import type { SaleCustomer } from '../useSaleCustomer';
import { useSendSale } from './useSendSale';

type Options = {
  cart: CartState;
  /** Who the sale goes to; once sold, the next sale starts with a final consumer. */
  customer: SaleCustomer;
  /** Where the sale happens; without it, nothing is sent. */
  locationId: string | undefined;
  /** Runs when the checkout closes: the focus goes back to the search. */
  onClose: () => void;
};

/** The checkout of the cart: open it, send the sale and show its result. */
export function useCheckout({ cart, customer, locationId, onClose }: Options) {
  const refreshCash = useRefreshCurrentCash();
  const [open, setOpen] = useState(false);
  const [sale, setSale] = useState<Sale>();
  // Who the confirmed sale went to, for its receipt: the POS's customer is already the next sale's.
  const [soldTo, setSoldTo] = useState<ChosenCustomer | null>(null);
  const [cashLost, setCashLost] = useState(false);
  const sending = useSendSale({
    cart,
    buyer: customer.customer,
    locationId,
    onSold: (confirmed, buyer) => {
      setSale(confirmed);
      setSoldTo(buyer);
      cart.change({ type: 'clear' });
      customer.setCustomer(null);
    },
    onCashLost: () => setCashLost(true),
  });

  function openCheckout() {
    sending.reset();
    setSale(undefined);
    setOpen(true);
  }

  function close() {
    setOpen(false);
    if (cashLost) {
      setCashLost(false);
      refreshCash();
    }
    if (sale) setSale(undefined);
    sending.reset();
    onClose();
  }

  return {
    open,
    sale,
    soldTo,
    openCheckout,
    onOpenChange: (next: boolean) => (next ? setOpen(true) : close()),
    close,
    confirm: sending.send,
    authorization: sending.authorization,
    pending: sending.pending,
    error: sending.error,
  };
}

export type Checkout = ReturnType<typeof useCheckout>;
