import { useParams } from 'react-router';
import { paths } from '@/app/routes/paths';
import { QueryState } from '@/shared/query';
import { ButtonLink, Inline, Page, Spinner, Stack } from '@/shared/ui';
import { usePurchase } from '../../api/purchases';
import { PurchaseTotals } from '../../components/PurchaseTotals/PurchaseTotals';
import { InvoiceSummary } from './components/InvoiceSummary/InvoiceSummary';
import { PurchaseActions } from './components/PurchaseActions/PurchaseActions';
import { PurchaseLines } from './components/PurchaseLines/PurchaseLines';
import { SupplierPayments } from './components/SupplierPayments/SupplierPayments';
import { VoidedNotice } from './components/VoidedNotice/VoidedNotice';

/** A registered purchase at /compras/:purchaseId: its invoice, its lines, what is owed and what was paid. */
export function PurchaseDetailPage() {
  const { purchaseId } = useParams();
  const purchase = usePurchase(purchaseId);
  const actions = (
    <Inline wrap>
      {purchase.data && <PurchaseActions purchase={purchase.data} />}
      <ButtonLink to={paths.purchases.list()} variant="secondary">
        Todas las compras
      </ButtonLink>
    </Inline>
  );
  return (
    <Page title={purchase.data ? `Compra ${purchase.data.invoice_number}` : 'Compra'} actions={actions}>
      <QueryState
        queries={[purchase]}
        errorTitle="No pudimos abrir la compra"
        loading={<Spinner label="Cargando la compra…" />}
      >
        {() =>
          purchase.data && (
            <Stack gap="6">
              <VoidedNotice purchase={purchase.data} />
              <InvoiceSummary purchase={purchase.data} />
              <PurchaseLines lines={purchase.data.lines} />
              <PurchaseTotals {...purchase.data} balanceDue={purchase.data.balance_due} />
              {purchase.data.payment === 'credit' && <SupplierPayments purchase={purchase.data} />}
            </Stack>
          )
        }
      </QueryState>
    </Page>
  );
}
