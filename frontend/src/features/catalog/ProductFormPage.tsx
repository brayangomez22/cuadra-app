import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useMemo, useState } from 'react';
import { Controller, useForm, useWatch, type FieldPath } from 'react-hook-form';
import { useNavigate, useParams } from 'react-router';
import { useAuth } from '@/features/auth';
import { api } from '@/shared/api/client';
import { ApiError, describeError, unwrap } from '@/shared/api/errors';
import { toDecimalInput } from '@/shared/format';
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
import {
  catalogKeys,
  useAttributeDefinitions,
  useCategories,
  useProduct,
  useUnits,
  useVariantAxes,
  type AttributeDefinition,
  type Category,
  type Product,
  type ProductInput,
  type UnitOfMeasure,
  type VariantAxis,
} from './api';
import { AttributeFields } from './AttributeFields';
import { VariantEditor, VariantList } from './ProductVariants';
import { VariantUnits } from './VariantUnits';
import { resolveRows, saveVariants, type ResolvedRow } from './saveVariants';
import { taxRateOptions } from './taxRates';
import {
  NO_CATEGORY,
  NO_VALUE,
  productSchema,
  type ProductFormInput,
  type ProductFormValues,
  type VariantRow,
} from './schemas';
import { skuPrefix } from './variantMatrix';
import { defaultVariant, hasAxes } from './variants';
import styles from './ProductFormPage.module.scss';

type VariantField = 'sku' | 'barcode' | 'cost' | 'price';

// Error codes of the API (api/openapi.yaml) that belong to one field. The
// ones of the default variant go to its row when the product has variants.
const fieldByErrorCode: Record<
  string,
  VariantField | 'name' | 'description' | 'category_id' | 'base_unit_id' | 'tax_rate'
> = {
  invalid_sku: 'sku',
  sku_taken: 'sku',
  invalid_barcode: 'barcode',
  barcode_taken: 'barcode',
  invalid_product_name: 'name',
  invalid_description: 'description',
  category_not_found: 'category_id',
  unit_not_found: 'base_unit_id',
  unit_required: 'base_unit_id',
  invalid_cost: 'cost',
  invalid_price: 'price',
  product_inactive: 'price',
  invalid_tax_rate: 'tax_rate',
};

// Errors of one attribute name it in details.attribute.
const attributeErrorCodes = new Set(['unknown_attribute', 'invalid_attribute_value', 'attribute_required']);

/** Creates a product at /productos/nuevo and edits one at /productos/:productId/editar. */
export function ProductFormPage() {
  const { productId } = useParams();
  const auth = useAuth();
  const user = auth.status === 'authenticated' ? auth.user : undefined;
  const variantsOn = user?.modules.includes('variants') ?? false;
  const unitsOn = user?.modules.includes('unit_conversions') ?? false;
  const product = useProduct(productId);
  const categories = useCategories();
  const units = useUnits();
  const definitions = useAttributeDefinitions();
  const axes = useVariantAxes({ enabled: variantsOn });
  const title = productId ? 'Editar producto' : 'Nuevo producto';
  const lists = [categories, units, definitions, ...(variantsOn ? [axes] : [])];

  if ((productId && product.isPending) || lists.some((query) => query.isPending)) {
    return (
      <Page title={title}>
        <Spinner label="Cargando el producto…" className={styles.loading} />
      </Page>
    );
  }
  const failed = [product, ...lists].find((query) => query.isError);
  if (failed || !categories.data || !units.data || !definitions.data) {
    return (
      <Page title={title}>
        <Alert tone="danger" title="No pudimos abrir el producto">
          {failed?.error?.message}
        </Alert>
      </Page>
    );
  }
  // key: a different product starts a fresh form.
  return (
    <ProductForm
      key={productId ?? 'new'}
      product={product.data}
      categories={categories.data}
      units={units.data}
      definitions={definitions.data}
      axes={variantsOn ? axes.data : undefined}
      alternateUnits={unitsOn}
      defaultTaxRate={user?.default_tax_rate ?? '0.19'}
    />
  );
}

type FormProps = {
  product?: Product;
  categories: Category[];
  units: UnitOfMeasure[];
  definitions: AttributeDefinition[];
  /** The tenant's variant axes, when the module variants is on. */
  axes?: VariantAxis[];
  /** Whether the module unit_conversions is on: the variants' alternate units are shown. */
  alternateUnits: boolean;
  /** The IVA rate a new product starts with: the tenant's. */
  defaultTaxRate: string;
};

function ProductForm({ product, categories, units, definitions, axes, alternateUnits, defaultTaxRate }: FormProps) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { toast } = useToast();
  const [formError, setFormError] = useState<string | null>(null);
  // A new product whose variants failed in part exists already: saving again edits it.
  const [created, setCreated] = useState<Product>();
  const [failedVariants, setFailedVariants] = useState(0);
  const schema = useMemo(() => productSchema(definitions), [definitions]);
  const {
    register,
    control,
    handleSubmit,
    setError,
    setValue,
    formState: { errors, isSubmitting },
  } = useForm<ProductFormInput, unknown, ProductFormValues>({
    resolver: zodResolver(schema),
    defaultValues: defaults({ product, units, definitions, axes, defaultTaxRate }),
  });
  const rows = useWatch({ control, name: 'variants' });

  const status = useMutation({
    mutationFn: async ({ id, active }: { id: string; active: boolean }) => {
      const path = active ? '/api/v1/products/{productId}/activate' : '/api/v1/products/{productId}/deactivate';
      return unwrap(await api.POST(path, { params: { path: { productId: id } } }));
    },
    onSuccess: (updated) => {
      queryClient.setQueryData(catalogKeys.product(updated.id), updated);
      setCreated((previous) => previous && updated);
      void queryClient.invalidateQueries({ queryKey: catalogKeys.lists });
      toast({ tone: 'success', title: updated.active ? 'Producto activado' : 'Producto desactivado' });
    },
    onError: (error) =>
      toast({ tone: 'danger', title: 'No pudimos cambiar el estado', description: describeError(error).message }),
  });

  // After (de)activating, the page re-renders with the product from the cache.
  const current = created ?? product;
  const inactive = current?.active === false;
  // With the module variants on, the variants the person picked; without it,
  // a product that already has them shows them.
  const matrix = axes !== undefined && rows.length > 0;
  const showVariants = axes !== undefined || (current !== undefined && hasAxes(current));

  async function save(values: ProductFormValues) {
    setFormError(null);
    setFailedVariants(0);
    let resolved: ResolvedRow[] = [];
    let defaultIndex = -1;
    if (matrix) {
      const result = resolveRows(values.variants, { sku: values.sku, cost: values.cost, price: values.price });
      if (result.errors.length > 0) {
        result.errors.forEach(({ index, field, message }, i) =>
          setError(`variants.${index}.${field}`, { message }, { shouldFocus: i === 0 }),
        );
        return;
      }
      resolved = result.rows;
      defaultIndex = defaultRow(values.variants, current);
    }
    const main = resolved[defaultIndex];
    const body = toInput(values, {
      sku: main?.sku ?? values.sku,
      barcode: main ? main.barcode : values.barcode,
      cost: main?.cost ?? values.cost,
      // An inactive product keeps its price (the API rejects a change until it is reactivated).
      price: inactive && current ? defaultVariant(current).price : (main?.price ?? values.price),
    });

    let saved: Product;
    try {
      saved = unwrap(
        current
          ? await api.PUT('/api/v1/products/{productId}', { params: { path: { productId: current.id } }, body })
          : await api.POST('/api/v1/products', { body }),
      );
    } catch (error) {
      const { code, message } = describeError(error);
      const attribute = error instanceof ApiError ? error.details.attribute : undefined;
      const field = fieldByErrorCode[code ?? ''];
      if (attribute && attributeErrorCodes.has(code ?? '') && definitions.some((d) => d.key === attribute)) {
        setError(`attributes.${attribute}`, { message }, { shouldFocus: true });
      } else if (field) {
        const isVariantField = field === 'sku' || field === 'barcode' || field === 'cost' || field === 'price';
        const path: FieldPath<ProductFormInput> =
          matrix && isVariantField ? `variants.${defaultIndex}.${field}` : field;
        setError(path, { message }, { shouldFocus: true });
      } else {
        setFormError(message);
      }
      return;
    }

    if (matrix) {
      const result = await saveVariants(saved, resolved, defaultIndex);
      saved = result.product;
      result.variantIds.forEach((id, index) => {
        if (id) setValue(`variants.${index}.variantId`, id);
      });
      if (result.errors.length > 0) {
        queryClient.setQueryData(catalogKeys.product(saved.id), saved);
        void queryClient.invalidateQueries({ queryKey: catalogKeys.lists });
        setCreated(saved);
        setFailedVariants(result.errors.length);
        result.errors.forEach(({ index, field, message }, i) =>
          setError(`variants.${index}.${field}`, { message }, { shouldFocus: i === 0 }),
        );
        return;
      }
    }
    queryClient.setQueryData(catalogKeys.product(saved.id), saved);
    void queryClient.invalidateQueries({ queryKey: catalogKeys.lists });
    toast({ tone: 'success', title: 'Producto guardado', description: saved.name });
    void navigate('/productos');
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
        {failedVariants > 0 && (
          <Alert tone="warning" title={`El producto se guardó, pero ${variantsCount(failedVariants)} no`}>
            Corrija lo marcado en la tabla de variantes y guarde de nuevo: solo se envía lo que falta.
          </Alert>
        )}

        <section className={styles.section} aria-labelledby="product-identity">
          <h2 id="product-identity" className={styles.sectionTitle}>
            Datos del producto
          </h2>
          <div className={styles.grid}>
            <Field
              label="SKU"
              hint={
                matrix
                  ? 'La base del SKU de cada variante. Se guarda en mayúsculas.'
                  : 'Su código interno. Se guarda en mayúsculas.'
              }
              error={errors.sku?.message}
            >
              <Input autoComplete="off" autoCapitalize="characters" spellCheck={false} {...register('sku')} />
            </Field>
            {!matrix && (
              <Field label="Código de barras" optional hint="Puede escanearlo aquí." error={errors.barcode?.message}>
                <Input autoComplete="off" spellCheck={false} {...register('barcode')} />
              </Field>
            )}
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
            <Field label="Unidad" hint="En la que cuenta y vende el producto." error={errors.base_unit_id?.message}>
              <Controller
                control={control}
                name="base_unit_id"
                render={({ field }) => (
                  <Select
                    options={units.map((unit) => ({ value: unit.id, label: `${unit.name} (${unit.code})` }))}
                    value={field.value || undefined}
                    onValueChange={field.onChange}
                    placeholder="Elija la unidad"
                  />
                )}
              />
            </Field>
          </div>
        </section>

        {definitions.length > 0 && (
          <section className={styles.section} aria-labelledby="product-attributes">
            <h2 id="product-attributes" className={styles.sectionTitle}>
              Atributos
            </h2>
            <AttributeFields
              definitions={definitions}
              register={register}
              control={control}
              errors={errors.attributes}
              className={styles.grid}
              wideClassName={styles.wide}
            />
          </section>
        )}

        <section className={styles.section} aria-labelledby="product-price">
          <h2 id="product-price" className={styles.sectionTitle}>
            Precio
          </h2>
          <div className={styles.grid}>
            <Field
              label="Costo"
              hint={matrix ? 'Sin IVA. Lo usan las variantes que no tengan el suyo.' : 'Lo que le cuesta, sin IVA.'}
              error={errors.cost?.message}
            >
              <Input inputMode="decimal" align="end" autoComplete="off" {...register('cost')} />
            </Field>
            <Field
              label="Precio de venta (sin IVA)"
              hint={
                inactive
                  ? 'Active el producto para cambiar el precio.'
                  : matrix
                    ? 'Lo usan las variantes que no tengan el suyo.'
                    : 'Use coma para los centavos: 12500,50.'
              }
              error={errors.price?.message}
            >
              <Input inputMode="decimal" align="end" autoComplete="off" disabled={inactive} {...register('price')} />
            </Field>
            <Field label="IVA" error={errors.tax_rate?.message}>
              <Controller
                control={control}
                name="tax_rate"
                render={({ field }) => (
                  <Select options={taxRateOptions(field.value)} value={field.value} onValueChange={field.onChange} />
                )}
              />
            </Field>
          </div>
        </section>

        {showVariants && (
          <section className={styles.section} aria-labelledby="product-variants">
            <h2 id="product-variants" className={styles.sectionTitle}>
              Variantes
            </h2>
            {axes !== undefined ? (
              <>
                <p className={styles.sectionHint}>
                  Si el producto se vende en varias versiones, elija sus valores: cada combinación es una variante con
                  su propio SKU, código y precio.
                </p>
                <VariantEditor
                  axes={axes}
                  control={control}
                  register={register}
                  errors={errors.variants}
                  inactive={inactive}
                />
              </>
            ) : (
              current && <VariantList product={current} />
            )}
          </section>
        )}

        {alternateUnits && (
          <section className={styles.section} aria-labelledby="product-units">
            <h2 id="product-units" className={styles.sectionTitle}>
              Unidades de venta
            </h2>
            <VariantUnits product={current} units={units} />
          </section>
        )}

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

function variantsCount(n: number) {
  return n === 1 ? '1 variante' : `${n} variantes`;
}

type DefaultsInput = {
  product?: Product;
  units: UnitOfMeasure[];
  definitions: AttributeDefinition[];
  axes?: VariantAxis[];
  defaultTaxRate: string;
};

function defaults({ product, units, definitions, axes, defaultTaxRate }: DefaultsInput): ProductFormInput {
  const main = product && defaultVariant(product);
  // With the module on, a product with variants edits them in the matrix.
  const matrix = axes !== undefined && product !== undefined && hasAxes(product);
  const unit = units.find((u) => u.code === 'und') ?? units[0];
  return {
    sku: matrix ? skuPrefix(product) : (main?.sku ?? ''),
    barcode: main?.barcode ?? '',
    name: product?.name ?? '',
    description: product?.description ?? '',
    category_id: product?.category_id ?? NO_CATEGORY,
    base_unit_id: product?.base_unit_id ?? unit?.id ?? '',
    tax_rate: product?.tax_rate ?? defaultTaxRate,
    // A decimal comma, as the person types it: "1.125" would read as a thousands dot.
    cost: main?.cost ? toDecimalInput(main.cost) : '',
    price: main ? toDecimalInput(main.price) : '',
    attributes: Object.fromEntries(
      definitions.map((definition) => [
        definition.key,
        attributeDefault(definition, product?.attributes[definition.key]),
      ]),
    ),
    variants: matrix
      ? product.variants.map((variant): VariantRow => ({
          variantId: variant.id,
          values: variant.axis_values,
          sku: variant.sku,
          barcode: variant.barcode ?? '',
          cost: variant.cost ? toDecimalInput(variant.cost) : '',
          price: toDecimalInput(variant.price),
          active: variant.active,
        }))
      : [],
  };
}

function attributeDefault(definition: AttributeDefinition, value: string | boolean | undefined): string | boolean {
  switch (definition.type) {
    case 'boolean':
      return value === true;
    case 'number':
      return typeof value === 'string' ? toDecimalInput(value) : '';
    case 'list':
      return typeof value === 'string' ? value : definition.required ? '' : NO_VALUE;
    case 'text':
      return typeof value === 'string' ? value : '';
  }
}

// defaultRow is the row of the product's default variant; for a new product
// (or one without variants yet), the first combination.
function defaultRow(rows: VariantRow[], product: Product | undefined) {
  const index = product ? rows.findIndex((row) => row.variantId === defaultVariant(product).id) : -1;
  return Math.max(index, 0);
}

type DefaultVariantInput = Pick<ProductInput, 'sku' | 'cost' | 'price'> & { barcode: string };

function toInput(values: ProductFormValues, variant: DefaultVariantInput): ProductInput {
  return {
    sku: variant.sku,
    ...(variant.barcode && { barcode: variant.barcode }),
    name: values.name,
    ...(values.description && { description: values.description }),
    category_id: values.category_id === NO_CATEGORY ? null : values.category_id,
    base_unit_id: values.base_unit_id,
    cost: variant.cost,
    price: variant.price,
    tax_rate: values.tax_rate,
    // The values replace the product's: an optional attribute left empty has none.
    attributes: Object.fromEntries(Object.entries(values.attributes).filter(([, value]) => value !== '')),
  };
}

// A subcategory shows its path: "Categoría › Subcategoría".
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
