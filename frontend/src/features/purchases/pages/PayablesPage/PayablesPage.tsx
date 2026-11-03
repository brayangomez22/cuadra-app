import { paths } from '@/app/routes/paths';
import { useSearchParamsState, pageParam } from '@/shared/hooks/useSearchParamsState';
import { QueryState } from '@/shared/query';
import { ButtonLink, EmptyState, IconTruck, Page, Pagination, Spinner, Stack } from '@/shared/ui';
import { PAYABLES_PAGE_SIZE, usePayables } from '../../api/supplierPayments';
import { PayablesSummary } from './components/PayablesSummary/PayablesSummary';
import { PayablesTable } from './components/PayablesTable/PayablesTable';

/** /compras/por-pagar: what is owed to each supplier, the most overdue first. */
export function PayablesPage() {
  const [page, setPage] = useSearchParamsState('pagina', pageParam);
  const offset = (page - 1) * PAYABLES_PAGE_SIZE;
  const payables = usePayables(offset);
  return (
    <Page
      title="Cuentas por pagar"
      description="Lo que se debe a cada proveedor por sus compras a crédito, y lo que ya venció."
      actions={
        <ButtonLink to={paths.purchases.list()} variant="secondary">
          Todas las compras
        </ButtonLink>
      }
    >
      <QueryState
        queries={[payables]}
        errorTitle="No pudimos cargar lo que se debe"
        loading={<Spinner label="Cargando lo que se debe…" />}
        isEmpty={payables.data?.items.length === 0}
        empty={
          <EmptyState
            icon={<IconTruck />}
            title="No se le debe nada a ningún proveedor"
            description="Las compras a crédito aparecen aquí hasta que se pagan."
          />
        }
      >
        {() =>
          payables.data && (
            <Stack gap="6">
              <PayablesSummary totals={payables.data.totals} />
              <PayablesTable items={payables.data.items} />
              <Pagination
                label="Páginas de proveedores"
                total={payables.data.total}
                limit={PAYABLES_PAGE_SIZE}
                offset={offset}
                onOffsetChange={(next) => setPage(next / PAYABLES_PAGE_SIZE + 1)}
              />
            </Stack>
          )
        }
      </QueryState>
    </Page>
  );
}
