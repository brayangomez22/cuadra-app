import { useParams } from 'react-router';
import { QueryState } from '@/shared/query';
import { Page, Spinner } from '@/shared/ui';
import { CustomerForm } from './components/CustomerForm/CustomerForm';
import { useCustomerFormData } from './hooks/useCustomerFormData';

/** Creates a customer at /clientes/nuevo and edits one at /clientes/:customerId/editar. */
export function CustomerFormPage() {
  const { customerId } = useParams();
  const { queries, data } = useCustomerFormData(customerId);
  // key: a different customer starts a fresh form.
  if (data) return <CustomerForm key={customerId ?? 'new'} data={data} />;
  return (
    <Page title={customerId ? 'Editar cliente' : 'Nuevo cliente'}>
      <QueryState
        queries={queries}
        errorTitle="No pudimos abrir el cliente"
        loading={<Spinner label="Cargando el cliente…" />}
      >
        {null}
      </QueryState>
    </Page>
  );
}
