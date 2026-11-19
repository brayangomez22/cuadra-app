import { useCan } from '@/features/auth';
import { QueryState } from '@/shared/query';
import { Page, Pagination, Spinner } from '@/shared/ui';
import { PAGE_SIZE, useCustomers } from '../../api/customers';
import { useCustomerSegments } from '../../api/segments';
import { CustomerFilters } from './components/CustomerFilters/CustomerFilters';
import { CustomerTable } from './components/CustomerTable/CustomerTable';
import { NoCustomers } from './components/NoCustomers/NoCustomers';
import { useCustomerListParams } from './hooks/useCustomerListParams';
import { CustomerListActions } from './components/CustomerListActions/CustomerListActions';
import styles from './CustomerListPage.module.scss';

/** The customers at /clientes: searched by name or document, filtered by segment and status, and paged. */
export function CustomerListPage() {
  const can = useCan();
  const canWrite = can('customers:write');
  const canReadCredit = can('receivables:read');
  const params = useCustomerListParams();
  const segments = useCustomerSegments();
  const customers = useCustomers(params.query);
  return (
    <Page
      title="Clientes"
      description="A quién le vende: documento, contacto y segmento, para la factura electrónica."
      actions={<CustomerListActions canWrite={canWrite} canReadCredit={canReadCredit} />}
    >
      <CustomerFilters params={params} segments={segments.data ?? []} />
      <QueryState
        queries={[customers]}
        errorTitle="No pudimos cargar los clientes"
        loading={<Spinner label="Cargando clientes…" className={styles.loading} />}
        isEmpty={customers.data?.items.length === 0}
        empty={<NoCustomers q={params.q} filtered={params.filtered} canWrite={canWrite} />}
      >
        {() =>
          customers.data && (
            <div className={styles.results} aria-busy={customers.isPlaceholderData || undefined}>
              <CustomerTable customers={customers.data.items} segments={segments.data ?? []} canWrite={canWrite} />
              <Pagination
                label="Páginas de clientes"
                total={customers.data.total}
                limit={PAGE_SIZE}
                offset={params.offset}
                onOffsetChange={params.setOffset}
              />
            </div>
          )
        }
      </QueryState>
    </Page>
  );
}
