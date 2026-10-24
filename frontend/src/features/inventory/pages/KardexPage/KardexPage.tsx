import { useParams } from 'react-router';
import { useProduct } from '@/features/catalog';
import { QueryState } from '@/shared/query';
import { Page, Spinner } from '@/shared/ui';
import { ProductKardex } from './components/ProductKardex/ProductKardex';
import styles from './KardexPage.module.scss';

/**
 * /productos/:productId/kardex: the movements of a variant of the product
 * (?variante=, by default its default variant), newest first, optionally of
 * one location (?sede=).
 */
export function KardexPage() {
  const { productId = '' } = useParams();
  const product = useProduct(productId);

  if (!product.isSuccess) {
    return (
      <Page title="Kardex">
        <QueryState
          queries={[product]}
          errorTitle="No pudimos abrir el producto"
          loading={<Spinner label="Cargando el producto…" className={styles.loading} />}
        >
          {null}
        </QueryState>
      </Page>
    );
  }
  return <ProductKardex product={product.data} />;
}
