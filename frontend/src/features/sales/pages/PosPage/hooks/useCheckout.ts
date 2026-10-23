import { useState } from 'react';
import { ApiError, describeError } from '@/shared/api/errors';
import { useConfirmSale, useRefreshCurrentCash, type Sale } from '../../../api';
import type { PaymentInput } from '../../../model/paymentMethods';
import type { CartState } from './useCart';

// Errors of a sale that mean the cashier's drawer is gone: the screen asks
// for it again once the checkout closes.
const cashSessionErrors = new Set(['cash_session_required', 'cash_session_closed']);

type Options = {
  cart: CartState;
  /** Where the sale happens; without it, nothing is sent. */
  locationId: string | undefined;
  /** Runs when the checkout closes: the focus goes back to the search. */
  onClose: () => void;
};

/** The checkout of the cart: open it, send the sale and show its result. */
export function useCheckout({ cart, locationId, onClose }: Options) {
  const confirmSale = useConfirmSale();
  const refreshCash = useRefreshCurrentCash();
  const [open, setOpen] = useState(false);
  const [sale, setSale] = useState<Sale>();
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

  function onError(error: Error) {
    if (!(error instanceof ApiError)) return;
    if (error.code === 'insufficient_stock') cart.setOutOfStock(error.details.variant_id);
    if (cashSessionErrors.has(error.code ?? '')) setCashLost(true);
  }

  function confirm(payments: PaymentInput[]) {
    if (!locationId) return;
    const lines = cart.totals.lines.flatMap((l) => (l.input ? [l.input] : []));
    const onSuccess = (confirmed: Sale) => {
      setSale(confirmed);
      cart.change({ type: 'clear' });
    };
    confirmSale.mutate({ location_id: locationId, lines, payments }, { onSuccess, onError });
  }

  return {
    open,
    sale,
    openCheckout,
    onOpenChange: (next: boolean) => (next ? setOpen(true) : close()),
    close,
    confirm,
    pending: confirmSale.isPending,
    error: confirmSale.isError ? describeError(confirmSale.error).message : undefined,
  };
}

export type Checkout = ReturnType<typeof useCheckout>;
