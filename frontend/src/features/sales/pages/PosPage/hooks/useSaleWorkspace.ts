import { useRef, useState } from 'react';
import { useAuth, useCan } from '@/features/auth';
import type { DiscountLimit } from '../../../model/cart/discount';
import type { CashSessionGuard } from './useCashSessionGuard';
import { useCart } from './useCart';
import { useCheckout } from './checkout/useCheckout';
import { usePosShortcuts } from './usePosShortcuts';
import { useSaleCredit } from './checkout/useSaleCredit';
import { useSaleCustomer } from './useSaleCustomer';

/**
 * The state of the sale in progress: the cart and its customer (kept per
 * user in this tab), their credit, the checkout, the cancel dialog and the
 * screen's shortcuts. Every dialog gives the focus back to the search.
 */
export function useSaleWorkspace(guard: CashSessionGuard, limit: DiscountLimit) {
  const auth = useAuth();
  const user = auth.status === 'authenticated' ? auth.user : undefined;
  const cartKey = `cuadra.pos.cart.${user?.id ?? ''}`;
  const cart = useCart(cartKey, limit);
  const sale = useSaleCustomer(cartKey);
  const credit = useSaleCredit(sale.customer);
  const canPickCustomer = useCan()('customers:read');
  const search = useRef<HTMLInputElement>(null);
  const focusSearch = () => requestAnimationFrame(() => search.current?.focus());
  const checkout = useCheckout({ cart, customer: sale, locationId: guard.locationId, onClose: focusSearch });
  const [cancelOpen, setCancelOpen] = useState(false);
  const closeCancel = () => {
    setCancelOpen(false);
    focusSearch();
  };
  const hasLocation = Boolean(guard.locationId);
  usePosShortcuts({
    search,
    dialogOpen: checkout.open || cancelOpen || sale.picking,
    canCharge: cart.totals.valid && hasLocation,
    hasLines: cart.cart.lines.length > 0,
    onCharge: checkout.openCheckout,
    onCancel: () => setCancelOpen(true),
    onPickCustomer: canPickCustomer ? () => sale.setPicking(true) : undefined,
  });

  const cancel = {
    open: cancelOpen,
    onOpenChange: (open: boolean) => (open ? setCancelOpen(true) : closeCancel()),
    onCancelSale: () => {
      cart.change({ type: 'clear' });
      sale.setCustomer(null);
      closeCancel();
    },
  };
  return { user, cart, sale, credit, search, focusSearch, checkout, cancel, hasLocation };
}
