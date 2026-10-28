import { useState } from 'react';
import type { ChosenCustomer } from '@/features/customers';
import { ApiError, describeError } from '@/shared/api/errors';
import { useRefreshCurrentCash } from '../../../api/cashSessions';
import { useConfirmSale } from '../../../api/sales';
import type { Sale } from '../../../api/types';
import type { PaymentInput } from '../../../model/payments/paymentMethods';
import type { CartState } from './useCart';
import type { SaleCustomer } from './useSaleCustomer';

// Errors of a sale that mean the cashier's drawer is gone: the screen asks
// for it again once the checkout closes.
const cashSessionErrors = new Set(['cash_session_required', 'cash_session_closed']);

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
  const confirmSale = useConfirmSale();
  const refreshCash = useRefreshCurrentCash();
  const [open, setOpen] = useState(false);
  const [sale, setSale] = useState<Sale>();
  // Who the confirmed sale went to, for its receipt: the POS's customer is already the next sale's.
  const [soldTo, setSoldTo] = useState<ChosenCustomer | null>(null);
  const [cashLost, setCashLost] = useState(false);

  function openCheckout() {
    confirmSale.reset();
    setSale(undefined);
    setOpen(true);
  }

  function close() {
    setOpen(false);
    if (cashLost) {
      setCashLost(false);
      refreshCash();
    }
    if (sale) {
      setSale(undefined);
      confirmSale.reset();
    }
    onClose();
  }

  const onError = (error: Error) => markSaleError(error, cart, () => setCashLost(true));

  function confirm(payments: PaymentInput[]) {
    if (!locationId) return;
    const lines = cart.totals.lines.flatMap((l) => (l.input ? [l.input] : []));
    const buyer = customer.customer;
    const onSuccess = (confirmed: Sale) => {
      setSale(confirmed);
      setSoldTo(buyer);
      cart.change({ type: 'clear' });
      customer.setCustomer(null);
    };
    const body = { location_id: locationId, lines, payments, ...(buyer && { customer_id: buyer.id }) };
    confirmSale.mutate(body, { onSuccess, onError });
  }

  return {
    open,
    sale,
    soldTo,
    openCheckout,
    onOpenChange: (next: boolean) => (next ? setOpen(true) : close()),
    close,
    confirm,
    pending: confirmSale.isPending,
    error: confirmSale.isError ? describeError(confirmSale.error).message : undefined,
  };
}

export type Checkout = ReturnType<typeof useCheckout>;

// markSaleError points at what failed: the line without stock, or the drawer
// that is gone (asked again once the checkout closes).
function markSaleError(error: Error, cart: CartState, onCashLost: () => void) {
  if (!(error instanceof ApiError)) return;
  if (error.code === 'insufficient_stock') cart.setOutOfStock(error.details.variant_id);
  if (cashSessionErrors.has(error.code ?? '')) onCashLost();
}
