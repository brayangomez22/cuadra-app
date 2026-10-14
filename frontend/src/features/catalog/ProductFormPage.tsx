import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { Controller, useForm } from 'react-hook-form';
import { useNavigate, useParams } from 'react-router';
import { api } from '@/shared/api/client';
import { describeError, unwrap } from '@/shared/api/errors';
import { formatPercent, toDecimalInput } from '@/shared/format';
import {
  Alert,
  Badge,
  Button,
  ButtonLink,
  Field,
  Input,
  Page,
  Select,
  Spinner,
  Textarea,
  useToast,
  type SelectOption,
} from '@/shared/ui';
import { catalogKeys, useCategories, useProduct, type Category, type Product, type ProductInput } from './api';
import { NO_CATEGORY, productSchema, type ProductFormInput, type ProductFormValues } from './schemas';
import { unitOptions } from './units';
import styles from './ProductFormPage.module.scss';

// Error codes of the API (api/openapi.yaml) that belong to one field.
const fieldByErrorCode: Record<string, keyof ProductFormInput> = {
  invalid_sku: 'sku',
  sku_taken: 'sku',
  invalid_barcode: 'barcode',
  barcode_taken: 'barcode',
  invalid_product_name: 'name',
  invalid_description: 'description',
  category_not_found: 'category_id',
  invalid_cost: 'cost',
  invalid_price: 'price',
  product_inactive: 'price',
  invalid_tax_rate: 'tax_rate',
};

// The usual rates in Colombia; a product with another rate keeps its own.
const commonTaxRates = ['0', '0.05', '0.19'];

/** Creates a product at /productos/nuevo and edits one at /productos/:productId/editar. */
export function ProductFormPage() {
  const { productId } = useParams();
  const product = useProduct(productId);
  const categories = useCategories();
  const title = productId ? 'Editar producto' : 'Nuevo producto';

  if ((productId && product.isPending) || categories.isPending) {
    return (
      <Page title={title}>
        <Spinner label="Cargando el producto…" className={styles.loading} />
      </Page>
    );
  }
  if (product.isError || categories.isError) {
    const error = product.error ?? categories.error;
    return (
      <Page title={title}>
        <Alert tone="danger" title="No pudimos abrir el producto">
          {error?.message}
        </Alert>
      </Page>
    );
  }
  // key: a different product starts a fresh form.
  return <ProductForm key={productId ?? 'new'} product={product.data} categories={categories.data} />;
}

type FormProps = { product?: Product; categories: Category[] };

function ProductForm({ product, categories }: FormProps) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { toast } = useToast();
  const [formError, setFormError] = useState<string | null>(null);
  const {
    register,
    control,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<ProductFormInput, unknown, ProductFormValues>({
    resolver: zodResolver(productSchema),
    defaultValues: defaults(product),
  });

  const status = useMutation({
    mutationFn: async ({ id, active }: { id: string; active: boolean }) => {
      const path = active ? '/api/v1/products/{productId}/activate' : '/api/v1/products/{productId}/deactivate';
      return unwrap(await api.POST(path, { params: { path: { productId: id } } }));
    },
    onSuccess: (updated) => {
      queryClient.setQueryData(catalogKeys.product(updated.id), updated);
      void queryClient.invalidateQueries({ queryKey: catalogKeys.lists });
      toast({ tone: 'success', title: updated.active ? 'Producto activado' : 'Producto desactivado' });
    },
    onError: (error) => toast({ tone: 'danger', title: 'No pudimos cambiar el estado', description: describeError(error).message }),
  });

  // After (de)activating, the page re-renders with the product from the cache.
  const current = product;
  const inactive = current?.active === false;

  async function save(values: ProductFormValues) {
    setFormError(null);
    // An inactive product keeps its price (the API rejects a change until it is reactivated).
    const body = toInput(values, inactive && current ? current.price : values.price);
    try {
      const saved = unwrap(
        current
          ? await api.PUT('/api/v1/products/{productId}', { params: { path: { productId: current.id } }, body })
          : await api.POST('/api/v1/products', { body }),
      );
      queryClient.setQueryData(catalogKeys.product(saved.id), saved);
      void queryClient.invalidateQueries({ queryKey: catalogKeys.lists });
      toast({ tone: 'success', title: 'Producto guardado', description: saved.name });
      void navigate('/productos');
    } catch (error) {
      const { code, message } = describeError(error);
      const field = fieldByErrorCode[code ?? ''];
      if (field) setError(field, { message }, { shouldFocus: true });
      else setFormError(message);
    }
  }

  return (
    <Page
      title={current ? 'Editar producto' : 'Nuevo producto'}
      description={
        current && (
          <span className={styles.subtitle}>
            {current.name}
            {inactive && <Badge tone="warning">Inactivo</Badge>}
          </span>
        )
      }
      actions={
        current && (
          <Button
            variant="secondary"
            loading={status.isPending}
            onClick={() => status.mutate({ id: current.id, active: inactive })}
          >
            {inactive ? 'Activar' : 'Desactivar'}
          </Button>
        )
      }
    >
      <form className={styles.form} noValidate onSubmit={handleSubmit(save)}>
        {formError && <Alert tone="danger">{formError}</Alert>}

        <section className={styles.section} aria-labelledby="product-identity">
          <h2 id="product-identity" className={styles.sectionTitle}>
            Datos del producto
          </h2>
          <div className={styles.grid}>
            <Field label="SKU" hint="Su código interno. Se guarda en mayúsculas." error={errors.sku?.message}>
              <Input autoComplete="off" autoCapitalize="characters" spellCheck={false} {...register('sku')} />
            </Field>
            <Field label="Código de barras" optional hint="Puede escanearlo aquí." error={errors.barcode?.message}>
              <Input autoComplete="off" spellCheck={false} {...register('barcode')} />
            </Field>
            <Field label="Nombre" error={errors.name?.message} className={styles.wide}>
              <Input autoComplete="off" {...register('name')} />
            </Field>
            <Field label="Descripción" optional error={errors.description?.message} className={styles.wide}>
              <Textarea {...register('description')} />
            </Field>
            <Field label="Categoría" error={errors.category_id?.message}>
              <Controller
                control={control}
                name="category_id"
                render={({ field }) => (
                  <Select options={categoryOptions(categories)} value={field.value} onValueChange={field.onChange} />
                )}
              />
            </Field>
            <Field label="Unidad" hint="En la que cuenta y vende el producto." error={errors.base_unit?.message}>
              <Controller
                control={control}
                name="base_unit"
                render={({ field }) => (
                  <Select options={unitOptions} value={field.value} onValueChange={field.onChange} />
                )}
              />
            </Field>
          </div>
        </section>

        <section className={styles.section} aria-labelledby="product-price">
          <h2 id="product-price" className={styles.sectionTitle}>
            Precio
          </h2>
          <div className={styles.grid}>
            <Field label="Costo" hint="Lo que le cuesta, sin IVA." error={errors.cost?.message}>
              <Input inputMode="decimal" align="end" autoComplete="off" {...register('cost')} />
            </Field>
            <Field
              label="Precio de venta (sin IVA)"
              hint={inactive ? 'Active el producto para cambiar el precio.' : 'Use coma para los centavos: 12500,50.'}
              error={errors.price?.message}
            >
              <Input inputMode="decimal" align="end" autoComplete="off" disabled={inactive} {...register('price')} />
            </Field>
            <Field label="IVA" error={errors.tax_rate?.message}>
              <Controller
                control={control}
                name="tax_rate"
                render={({ field }) => (
                  <Select options={taxOptions(field.value)} value={field.value} onValueChange={field.onChange} />
                )}
              />
            </Field>
          </div>
        </section>

        <div className={styles.footer}>
          <ButtonLink to="/productos" variant="ghost">
            Cancelar
          </ButtonLink>
          <Button type="submit" size="lg" loading={isSubmitting}>
            Guardar producto
          </Button>
        </div>
      </form>
    </Page>
  );
}

function defaults(product?: Product): ProductFormInput {
  return {
    sku: product?.sku ?? '',
    barcode: product?.barcode ?? '',
    name: product?.name ?? '',
    description: product?.description ?? '',
    category_id: product?.category_id ?? NO_CATEGORY,
    base_unit: product?.base_unit ?? 'und',
    tax_rate: product?.tax_rate ?? '0.19',
    // A decimal comma, as the person types it: "1.125" would read as a thousands dot.
    cost: product?.cost ? toDecimalInput(product.cost) : '',
    price: product ? toDecimalInput(product.price) : '',
  };
}

function toInput(values: ProductFormValues, price: string): ProductInput {
  return {
    sku: values.sku,
    ...(values.barcode && { barcode: values.barcode }),
    name: values.name,
    ...(values.description && { description: values.description }),
    category_id: values.category_id === NO_CATEGORY ? null : values.category_id,
    base_unit: values.base_unit,
    cost: values.cost,
    price,
    tax_rate: values.tax_rate,
  };
}

// A subcategory shows its path: "Plomería › Tubería".
function categoryOptions(categories: Category[]): SelectOption[] {
  const byId = new Map(categories.map((category) => [category.id, category]));
  const path = (category: Category, depth = 0): string => {
    const parent = category.parent_id ? byId.get(category.parent_id) : undefined;
    return parent && depth < 10 ? `${path(parent, depth + 1)} › ${category.name}` : category.name;
  };
  const options = categories.map((category) => ({ value: category.id, label: path(category) }));
  options.sort((a, b) => a.label.localeCompare(b.label, 'es'));
  return [{ value: NO_CATEGORY, label: 'Sin categoría' }, ...options];
}

function taxOptions(current: string): SelectOption[] {
  const rates = commonTaxRates.includes(current) ? commonTaxRates : [...commonTaxRates, current];
  return rates.map((rate) => ({ value: rate, label: formatPercent(rate, 'rate') }));
}
