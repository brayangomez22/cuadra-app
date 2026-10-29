import { useParams } from 'react-router';
import { paths } from '@/app/routes/paths';
import { QueryState } from '@/shared/query';
import { ButtonLink, Page, Spinner, Stack } from '@/shared/ui';
import { usePurchase } from '../../api/purchases';
import { PurchaseTotals } from '../../components/PurchaseTotals/PurchaseTotals';
import { InvoiceSummary } from './components/InvoiceSummary/InvoiceSummary';
import { PurchaseLines } from './components/PurchaseLines/PurchaseLines';

/** A registered purchase at /compras/:purchaseId: its invoice, its lines and what is owed. */
export function PurchaseDetailPage() {
  const { purchaseId } = useParams();
  const purchase = usePurchase(purchaseId);
  const back = (
    <ButtonLink to={paths.purchases.list()} variant="secondary">
      Todas las compras
    </ButtonLink>
  );
  return (
    <Page title={purchase.data ? `Compra ${purchase.data.invoice_number}` : 'Compra'} actions={back}>
      <QueryState
        queries={[purchase]}
        errorTitle="No pudimos abrir la compra"
        loading={<Spinner label="Cargando la compra…" />}
      >
        {() =>
          purchase.data && (
            <Stack gap="6">
              <InvoiceSummary purchase={purchase.data} />
              <PurchaseLines lines={purchase.data.lines} />
              <PurchaseTotals {...purchase.data} balanceDue={purchase.data.balance_due} />
            </Stack>
          )
        }
      </QueryState>
    </Page>
  );
}
