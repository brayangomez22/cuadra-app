import { FormProvider, useWatch } from 'react-hook-form';
import { paths } from '@/app/routes/paths';
import { Badge, Button, ButtonLink, Page } from '@/shared/ui';
import { useProductForm } from '../../hooks/useProductForm';
import type { ProductFormData } from '../../hooks/useProductFormData';
import { useSaveProduct } from '../../hooks/useSaveProduct';
import { ProductSections } from '../ProductSections/ProductSections';
import { ProductStatusButton } from '../ProductStatusButton/ProductStatusButton';
import { SaveAlerts } from '../SaveAlerts/SaveAlerts';
import styles from './ProductForm.module.scss';

/** The product form: its sections, the status action and the save. */
export function ProductForm({ data }: { data: ProductFormData }) {
  const form = useProductForm(data);
  const rows = useWatch({ control: form.control, name: 'variants' });
  // Saving or (de)activating keeps the product the API answered in the cache, and data.product with it.
  const current = data.product;
  const save = useSaveProduct({ form, current, definitions: data.definitions, rows });
  return (
    <Page
      title={current ? 'Editar producto' : 'Nuevo producto'}
      description={
        current && (
          <span className={styles.subtitle}>
            {current.name}
            {!current.active && <Badge tone="warning">Inactivo</Badge>}
          </span>
        )
      }
      actions={current && <ProductStatusButton product={current} />}
    >
      <FormProvider {...form}>
        <form className={styles.root} noValidate onSubmit={(event) => void save.submit(event)}>
          <SaveAlerts formError={save.formError} failedVariants={save.failedVariants} />
          {/* With the module variants on, the variants the person picked. */}
          <ProductSections data={data} current={current} matrix={data.axes !== undefined && rows.length > 0} />
          <div className={styles.footer}>
            <ButtonLink to={paths.products.list()} variant="ghost">
              Cancelar
            </ButtonLink>
            <Button type="submit" size="lg" loading={form.formState.isSubmitting}>
              Guardar producto
            </Button>
          </div>
        </form>
      </FormProvider>
    </Page>
  );
}
