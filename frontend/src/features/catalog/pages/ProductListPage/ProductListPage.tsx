import { paths } from '@/app/routes/paths';
import { useCan } from '@/features/auth';
import { QueryState } from '@/shared/query';
import { ButtonLink, IconPlus, Page, Pagination, Spinner } from '@/shared/ui';
import { PAGE_SIZE, useProducts } from '../../api/products';
import { NoProducts } from './components/NoProducts/NoProducts';
import { ProductFilters } from './components/ProductFilters/ProductFilters';
import { ProductTable } from './components/ProductTable/ProductTable';
import { useProductListParams } from './hooks/useProductListParams';
import styles from './ProductListPage.module.scss';

/** The catalog at /productos: searched by name, SKU or barcode, filtered by status and paged. */
export function ProductListPage() {
  const can = useCan();
  const canWrite = can('catalog:write');
  const params = useProductListParams();
  const { q, status, offset } = params;
  const products = useProducts({ q: q || undefined, status, limit: PAGE_SIZE, offset });
  return (
    <Page
      title="Productos"
      description="Su catálogo: precios, IVA y unidades."
      actions={
        canWrite && (
          <ButtonLink to={paths.products.new()} icon={<IconPlus />}>
            Nuevo producto
          </ButtonLink>
        )
      }
    >
      <ProductFilters
        text={params.text}
        onTextChange={params.setText}
        status={status}
        onStatusChange={params.setStatus}
      />
      <QueryState
        queries={[products]}
        errorTitle="No pudimos cargar los productos"
        loading={<Spinner label="Cargando productos…" className={styles.loading} />}
        isEmpty={products.data?.items.length === 0}
        empty={<NoProducts q={q} status={status} canWrite={canWrite} />}
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
            </div>
          )
        }
      </QueryState>
    </Page>
  );
}
