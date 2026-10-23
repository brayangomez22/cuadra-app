import { useRef, useState } from 'react';
import { useAuth } from '@/features/auth';
import { Page } from '@/shared/ui';
import type { DiscountLimit } from '../../../../model/cart/discount';
import type { CashSessionGuard } from '../../hooks/useCashSessionGuard';
import { useCart } from '../../hooks/useCart';
import { useCheckout } from '../../hooks/useCheckout';
import { usePosShortcuts } from '../../hooks/usePosShortcuts';
import { CancelSaleDialog } from '../CancelSaleDialog/CancelSaleDialog';
import { SaleCart } from '../SaleCart/SaleCart';
import { SaleCheckout } from '../SaleCheckout/SaleCheckout';
import { SaleHeader } from '../SaleHeader/SaleHeader';
import { SaleSummary } from '../SaleSummary/SaleSummary';
import styles from './PosWorkspace.module.scss';

type Props = { guard: CashSessionGuard; limit: DiscountLimit };

/**
 * The sale in progress, once the POS can sell: search or scan, adjust the
 * cart and charge, all from the keyboard.
 */
export function PosWorkspace({ guard, limit }: Props) {
  const auth = useAuth();
  const user = auth.status === 'authenticated' ? auth.user : undefined;
  const cart = useCart(`cuadra.pos.cart.${user?.id ?? ''}`, limit);
  const search = useRef<HTMLInputElement>(null);
  const focusSearch = () => requestAnimationFrame(() => search.current?.focus());
  const checkout = useCheckout({ cart, locationId: guard.locationId, onClose: focusSearch });
  const [cancelOpen, setCancelOpen] = useState(false);
  const closeCancel = () => {
    setCancelOpen(false);
    focusSearch();
  };
  const hasLocation = Boolean(guard.locationId);
  usePosShortcuts({
    search,
    dialogOpen: checkout.open || cancelOpen,
    canCharge: cart.totals.valid && hasLocation,
    hasLines: cart.cart.lines.length > 0,
    onCharge: checkout.openCheckout,
    onCancel: () => setCancelOpen(true),
  });

  const hasHeader = Boolean(guard.session) || guard.activeLocations.length > 1;
  return (
    <Page
      title="Vender"
      className={styles.root}
      actions={hasHeader && <SaleHeader guard={guard} onChosen={focusSearch} />}
    >
      <div className={styles.workspace}>
        <SaleCart cart={cart} limit={limit} searchRef={search} onDone={focusSearch} user={user} />
        <SaleSummary totals={cart.totals} hasLocation={hasLocation} onCharge={checkout.openCheckout} />
      </div>
      <SaleCheckout checkout={checkout} total={cart.totals.total} locations={guard.locations} user={user} />
      <CancelSaleDialog
        open={cancelOpen}
        onOpenChange={(open) => (open ? setCancelOpen(true) : closeCancel())}
        onCancelSale={() => {
          cart.change({ type: 'clear' });
          closeCancel();
        }}
      />
    </Page>
  );
}
