import { paths } from '@/app/routes/paths';
import { useCan } from '@/features/auth';
import { QueryState } from '@/shared/query';
import { ButtonLink, IconPlus, Page, Pagination, Spinner } from '@/shared/ui';
import { SUPPLIER_PAGE_SIZE, useSuppliers } from '../../api/suppliers';
import { NoSuppliers } from './components/NoSuppliers/NoSuppliers';
import { SupplierFilters } from './components/SupplierFilters/SupplierFilters';
import { SupplierTable } from './components/SupplierTable/SupplierTable';
import { useSupplierListParams } from './hooks/useSupplierListParams';
import styles from './SupplierListPage.module.scss';

/** The suppliers at /compras/proveedores: searched by name or NIT, filtered by status, and paged. */
export function SupplierListPage() {
  const canWrite = useCan()('purchases:write');
  const params = useSupplierListParams();
  const suppliers = useSuppliers(params.query);
  return (
    <Page
      title="Proveedores"
      description="A quién le compra: su NIT, el plazo que le da y cuánto le debe."
      actions={
        canWrite && (
          <ButtonLink to={paths.purchases.newSupplier()} icon={<IconPlus />}>
            Nuevo proveedor
          </ButtonLink>
        )
      }
    >
      <SupplierFilters params={params} />
      <QueryState
        queries={[suppliers]}
        errorTitle="No pudimos cargar los proveedores"
        loading={<Spinner label="Cargando proveedores…" className={styles.loading} />}
        isEmpty={suppliers.data?.items.length === 0}
        empty={<NoSuppliers q={params.q} filtered={params.filtered} canWrite={canWrite} />}
      >
        {() =>
          suppliers.data && (
            <div className={styles.results} aria-busy={suppliers.isPlaceholderData || undefined}>
              <SupplierTable suppliers={suppliers.data.items} canWrite={canWrite} />
              <Pagination
                label="Páginas de proveedores"
                total={suppliers.data.total}
                limit={SUPPLIER_PAGE_SIZE}
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
