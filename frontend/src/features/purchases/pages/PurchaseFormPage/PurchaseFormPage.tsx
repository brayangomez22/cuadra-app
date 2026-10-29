import { QueryState } from '@/shared/query';
import { Page, Spinner } from '@/shared/ui';
import { PurchaseForm } from './components/PurchaseForm/PurchaseForm';
import { usePurchaseFormData } from './hooks/usePurchaseFormData';

/** Registers the invoice of a supplier at /compras/nueva. */
export function PurchaseFormPage() {
  const { queries, data } = usePurchaseFormData();
  if (data) return <PurchaseForm data={data} />;
  return (
    <Page title="Registrar compra">
      <QueryState
        queries={queries}
        errorTitle="No pudimos abrir el formulario de compra"
        loading={<Spinner label="Cargando proveedores y sedes…" />}
      >
        {null}
      </QueryState>
    </Page>
  );
}
