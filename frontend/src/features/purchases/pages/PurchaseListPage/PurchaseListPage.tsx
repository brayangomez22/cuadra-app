import { paths } from '@/app/routes/paths';
import { useCan } from '@/features/auth';
import { QueryState } from '@/shared/query';
import { ButtonLink, IconPlus, Inline, Page, Pagination, Spinner } from '@/shared/ui';
import { PURCHASE_PAGE_SIZE, usePurchases } from '../../api/purchases';
import { useSuppliers } from '../../api/suppliers';
import { NoPurchases } from './components/NoPurchases/NoPurchases';
import { PurchaseFilters } from './components/PurchaseFilters/PurchaseFilters';
import { PurchaseTable } from './components/PurchaseTable/PurchaseTable';
import { usePurchaseListParams } from './hooks/usePurchaseListParams';
import styles from './PurchaseListPage.module.scss';

// Every supplier, inactive too: their purchases are still listed.
const allSuppliers = { status: 'all', limit: 100 } as const;

/** The purchases at /compras: filtered by supplier, status and days, paged, with what is owed of each. */
export function PurchaseListPage() {
  const canWrite = useCan()('purchases:write');
  const params = usePurchaseListParams();
  const suppliers = useSuppliers(allSuppliers);
  const purchases = usePurchases(params.query);
  return (
    <Page
      title="Compras"
      description="Las facturas de sus proveedores: lo que entró al inventario y lo que falta pagar."
      actions={
        <Inline wrap>
          <ButtonLink to={paths.purchases.suppliers()} variant="secondary">
            Proveedores
          </ButtonLink>
          {canWrite && (
            <ButtonLink to={paths.purchases.new()} icon={<IconPlus />}>
              Registrar compra
            </ButtonLink>
          )}
        </Inline>
      }
    >
      <PurchaseFilters params={params} suppliers={suppliers.data?.items ?? []} />
      <QueryState
        queries={[purchases]}
        errorTitle="No pudimos cargar las compras"
        loading={<Spinner label="Cargando compras…" className={styles.loading} />}
        isEmpty={purchases.data?.items.length === 0}
        empty={<NoPurchases filtered={params.filtered} canWrite={canWrite} />}
      >
        {() =>
          purchases.data && (
            <div className={styles.results} aria-busy={purchases.isPlaceholderData || undefined}>
              <PurchaseTable purchases={purchases.data.items} />
              <Pagination
                label="Páginas de compras"
                total={purchases.data.total}
                limit={PURCHASE_PAGE_SIZE}
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
