import { useParams } from 'react-router';
import { QueryState } from '@/shared/query';
import { Page, Spinner } from '@/shared/ui';
import { ProductForm } from './components/ProductForm/ProductForm';
import { useProductFormData } from './hooks/useProductFormData';
import styles from './ProductFormPage.module.scss';

/** Creates a product at /productos/nuevo and edits one at /productos/:productId/editar. */
export function ProductFormPage() {
  const { productId } = useParams();
  const { queries, data } = useProductFormData(productId);
  // key: a different product starts a fresh form.
  if (data) return <ProductForm key={productId ?? 'new'} data={data} />;
  return (
    <Page title={productId ? 'Editar producto' : 'Nuevo producto'}>
      <QueryState
        queries={queries}
        errorTitle="No pudimos abrir el producto"
        loading={<Spinner label="Cargando el producto…" className={styles.loading} />}
      >
        {null}
      </QueryState>
    </Page>
  );
}
