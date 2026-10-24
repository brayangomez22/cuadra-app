import { QueryState } from '@/shared/query';
import { Page, Spinner } from '@/shared/ui';
import { useDiscountLimit } from '../../api/sales';
import type { DiscountLimit } from '../../model/cart/discount';
import { PosUnavailable } from './components/PosUnavailable/PosUnavailable';
import { PosWorkspace } from './components/PosWorkspace/PosWorkspace';
import { useCashSessionGuard } from './hooks/useCashSessionGuard';

// Until the API answers, nobody discounts: the safe default (docs/decisiones.md).
const noDiscount: DiscountLimit = { unlimited: false, max: '0' };

/**
 * The point of sale. F2 goes to the search, F4 charges and Escape cancels the
 * sale. Sales go to the cashier's open cash session, at its location; if the
 * tenant requires one and there is none, the screen asks to open it.
 */
export function PosPage() {
  const guard = useCashSessionGuard();
  // Asked here, with the locations and the drawer, so the three go out at once.
  const limit = useDiscountLimit();
  if (guard.status === 'ready') return <PosWorkspace guard={guard} limit={limit.data ?? noDiscount} />;
  const { status } = guard;
  return (
    <Page title="Vender">
      <QueryState
        queries={guard.queries}
        errorTitle="No pudimos abrir la caja"
        loading={<Spinner label="Cargando la caja…" />}
      >
        {() => status !== 'loading' && <PosUnavailable reason={status} />}
      </QueryState>
    </Page>
  );
}
