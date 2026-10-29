import { Page } from '@/shared/ui';
import type { DiscountLimit } from '../../../../model/cart/discount';
import type { CashSessionGuard } from '../../hooks/useCashSessionGuard';
import { useSaleWorkspace } from '../../hooks/useSaleWorkspace';
import { CancelSaleDialog } from '../CancelSaleDialog/CancelSaleDialog';
import { SaleCart } from '../SaleCart/SaleCart';
import { SaleCheckout } from '../SaleCheckout/SaleCheckout';
import { SaleCustomer } from '../SaleCustomer/SaleCustomer';
import { SaleHeader } from '../SaleHeader/SaleHeader';
import { SaleSummary } from '../SaleSummary/SaleSummary';
import styles from './PosWorkspace.module.scss';

type Props = { guard: CashSessionGuard; limit: DiscountLimit };

/**
 * The sale in progress, once the POS can sell: search or scan, adjust the
 * cart and charge, all from the keyboard.
 */
export function PosWorkspace({ guard, limit }: Props) {
  const { user, cart, sale, credit, search, focusSearch, checkout, cancel, hasLocation } = useSaleWorkspace(
    guard,
    limit,
  );
  const hasHeader = Boolean(guard.session) || guard.activeLocations.length > 1;
  return (
    <Page
      title="Vender"
      className={styles.root}
      actions={hasHeader && <SaleHeader guard={guard} onChosen={focusSearch} />}
    >
      <div className={styles.workspace}>
        <div className={styles.sale}>
          <SaleCustomer sale={sale} onDone={focusSearch} />
          <SaleCart cart={cart} limit={limit} searchRef={search} onDone={focusSearch} />
        </div>
        <SaleSummary totals={cart.totals} hasLocation={hasLocation} onCharge={checkout.openCheckout} />
      </div>
      <SaleCheckout
        checkout={checkout}
        total={cart.totals.total}
        credit={credit}
        locations={guard.locations}
        user={user}
      />
      <CancelSaleDialog {...cancel} />
    </Page>
  );
}
