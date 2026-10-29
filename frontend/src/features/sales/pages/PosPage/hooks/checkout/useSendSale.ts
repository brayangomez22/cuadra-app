import type { ChosenCustomer } from '@/features/customers';
import { ApiError, describeError } from '@/shared/api/errors';
import { useConfirmSale } from '../../../../api/sales';
import type { ConfirmSaleInput, Sale } from '../../../../api/types';
import type { PaymentInput } from '../../../../model/payments/paymentMethods';
import type { CartState } from '../useCart';
import { useCreditAuthorization } from './useCreditAuthorization';

// Errors of a sale that mean the cashier's drawer is gone: the screen asks
// for it again once the checkout closes.
const cashSessionErrors = new Set(['cash_session_required', 'cash_session_closed']);

type Credentials = NonNullable<ConfirmSaleInput['credit_authorization']>;

type Options = {
  cart: CartState;
  /** Who the sale goes to; null is a final consumer. */
  buyer: ChosenCustomer | null;
  /** Where the sale happens; without it, nothing is sent. */
  locationId: string | undefined;
  onSold: (sale: Sale, buyer: ChosenCustomer | null) => void;
  /** The drawer is gone: the screen asks for it again once the checkout closes. */
  onCashLost: () => void;
};

/**
 * Sends the cart as a sale. A credit sale over the customer's limit waits for
 * an administrator's authorization and is sent again with it.
 */
export function useSendSale({ cart, buyer, locationId, onSold, onCashLost }: Options) {
  const confirmSale = useConfirmSale();
  const authorization = useCreditAuthorization();

  /** Sends the sale; credit is the administrator's authorization to go over the customer's limit. */
  function send(payments: PaymentInput[], credit?: Credentials) {
    if (!locationId) return;
    const lines = cart.totals.lines.flatMap((l) => (l.input ? [l.input] : []));
    const body: ConfirmSaleInput = { location_id: locationId, lines, payments };
    if (buyer) body.customer_id = buyer.id;
    if (credit) body.credit_authorization = credit;
    const onSuccess = (sale: Sale) => {
      authorization.clear();
      onSold(sale, buyer);
    };
    const onError = (error: Error) => {
      if (!authorization.request(error, payments)) markSaleError(error, cart, onCashLost);
    };
    confirmSale.mutate(body, { onSuccess, onError });
  }

  function reset() {
    authorization.clear();
    confirmSale.reset();
  }

  return {
    send,
    reset,
    /** The credit sale waiting for an administrator's authorization, and how to send it or go back to the payments. */
    authorization: {
      pending: authorization.pending,
      authorize: (credit: Credentials) => {
        if (authorization.pending) send(authorization.pending.payments, credit);
      },
      cancel: reset,
    },
    pending: confirmSale.isPending,
    error: confirmSale.isError ? describeError(confirmSale.error).message : undefined,
  };
}

// markSaleError points at what failed: the line without stock, or the drawer that is gone.
function markSaleError(error: Error, cart: CartState, onCashLost: () => void) {
  if (!(error instanceof ApiError)) return;
  if (error.code === 'insufficient_stock') cart.setOutOfStock(error.details.variant_id);
  if (cashSessionErrors.has(error.code ?? '')) onCashLost();
}
