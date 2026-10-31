import { useState, type ComponentProps } from 'react';
import { useParams } from 'react-router';
import { paths } from '@/app/routes/paths';
import { useCan } from '@/features/auth';
import { QueryState } from '@/shared/query';
import { Button, ButtonLink, IconPrinter, Inline, Page, Spinner, Stack } from '@/shared/ui';
import { PaperSelect } from '../../components/PaperSelect/PaperSelect';
import { PrintableReceipt } from '../../components/PrintableReceipt/PrintableReceipt';
import { Receipt } from '../../components/Receipt/Receipt';
import { VoidedNotice } from './components/VoidedNotice/VoidedNotice';
import { VoidSaleDialog } from './components/VoidSaleDialog/VoidSaleDialog';
import { useSaleReceipt } from './hooks/useSaleReceipt';
import styles from './SaleDetailPage.module.scss';

/** A past sale at /ventas/:saleId: its receipt, to read and reprint, and voiding it. */
export function SaleDetailPage() {
  const { saleId = '' } = useParams();
  const { queries, receipt, choosePaper } = useSaleReceipt(saleId);
  const back = (
    <ButtonLink to={paths.sales.list()} variant="secondary">
      Todas las ventas
    </ButtonLink>
  );
  return (
    <Page title={receipt ? `Venta N.º ${receipt.sale.number}` : 'Venta'} actions={back}>
      <QueryState
        queries={queries}
        errorTitle="No pudimos abrir la venta"
        loading={<Spinner label="Cargando la venta…" className={styles.loading} />}
      >
        {() => receipt && <LoadedSale receipt={receipt} onPaperChange={choosePaper} />}
      </QueryState>
    </Page>
  );
}

type LoadedProps = {
  receipt: ComponentProps<typeof Receipt>;
  onPaperChange: ComponentProps<typeof PaperSelect>['onPaperChange'];
};

function LoadedSale({ receipt, onPaperChange }: LoadedProps) {
  const can = useCan();
  const [voiding, setVoiding] = useState(false);
  const { sale, paper } = receipt;
  return (
    <Stack gap="6">
      {sale.void && <VoidedNotice info={sale.void} />}
      <Inline wrap className={styles.actions}>
        <PaperSelect paper={paper} onPaperChange={onPaperChange} />
        <Button variant="secondary" icon={<IconPrinter />} onClick={() => window.print()}>
          Imprimir tirilla
        </Button>
        {sale.status === 'confirmed' && can('sales:void') && (
          <Button variant="danger" onClick={() => setVoiding(true)}>
            Anular venta
          </Button>
        )}
      </Inline>
      <div className={styles.paper}>
        <Receipt {...receipt} />
      </div>
      <PrintableReceipt {...receipt} />
      {voiding && <VoidSaleDialog sale={sale} onClose={() => setVoiding(false)} />}
    </Stack>
  );
}
