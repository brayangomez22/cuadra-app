import { useCan } from '@/features/auth';
import { QueryState } from '@/shared/query';
import { Page, Pagination, Spinner } from '@/shared/ui';
import { PAGE_SIZE, useProducts } from '../../api/products';
import { ProductListActions } from './components/ProductListActions/ProductListActions';
import { NoProducts } from './components/NoProducts/NoProducts';
import { PageJump } from './components/PageJump/PageJump';
import { ProductFilters } from './components/ProductFilters/ProductFilters';
import { ProductTable } from './components/ProductTable/ProductTable';
import { useProductListParams } from './hooks/useProductListParams';
import styles from './ProductListPage.module.scss';

/** The catalog at /productos: searched by name, SKU or barcode, filtered by status and paged. */
export function ProductListPage() {
  const can = useCan();
  const canWrite = can('catalog:write');
  const canImport = can('imports:run');
  const params = useProductListParams();
  const { q, status, offset, sort } = params;
  const category_id = params.category || undefined;
  const products = useProducts({ q: q || undefined, status, category_id, sort, limit: PAGE_SIZE, offset });
  return (
    <Page
      title="Productos"
      description="Su catálogo: precios, IVA y unidades."
      actions={(canWrite || canImport) && <ProductListActions canWrite={canWrite} canImport={canImport} />}
    >
      <ProductFilters params={params} canSeeMargin={canWrite} />
      <QueryState
        queries={[products]}
        errorTitle="No pudimos cargar los productos"
        loading={<Spinner label="Cargando productos…" className={styles.loading} />}
        isEmpty={products.data?.items.length === 0}
        empty={<NoProducts q={q} status={status} canWrite={canWrite} canImport={canImport} />}
      >
        {() =>
          products.data && (
            <div className={styles.results} aria-busy={products.isPlaceholderData || undefined}>
              <ProductTable products={products.data.items} canWrite={canWrite} canSeeStock={can('inventory:read')} />
              <Pagination
                label="Páginas de productos"
                total={products.data.total}
                limit={PAGE_SIZE}
                offset={offset}
                onOffsetChange={params.setOffset}
              />
              <PageJump
                total={products.data.total}
                limit={PAGE_SIZE}
                offset={offset}
                onOffsetChange={params.setOffset}
              />
            </div>
          )
        }
      </QueryState>
    </Page>
  );
}
