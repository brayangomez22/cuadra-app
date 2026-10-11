import { zodResolver } from '@hookform/resolvers/zod';
import { useQueryClient } from '@tanstack/react-query';
import { useEffect, useState } from 'react';
import { Controller, useForm, useWatch } from 'react-hook-form';
import { useSearchParams } from 'react-router';
import { z } from 'zod';
import { ProductPicker, unitShort, useProduct, type Product } from '@/features/catalog';
import { api } from '@/shared/api/client';
import { describeError, unwrap } from '@/shared/api/errors';
import { decimalField } from '@/shared/forms/decimalField';
import { formatQuantity, toDecimalInput } from '@/shared/format';
import {
  Alert,
  Button,
  ButtonLink,
  EmptyState,
  Field,
  IconBox,
  Input,
  Page,
  Select,
  Spinner,
  useToast,
} from '@/shared/ui';
import { inventoryKeys, useLocations } from './api';
import styles from './ReceiptPage.module.scss';

// Same limits as ReceiptInput in api/openapi.yaml.
const schema = z.object({
  product: z.custom<Product | null>().refine((product) => product !== null, 'Elija el producto'),
  location_id: z.string().min(1, 'Elija la sede'),
  quantity: decimalField('positive', {
    empty: 'Escriba la cantidad',
    belowMin: 'La cantidad debe ser mayor que cero',
  }),
  unit_cost: decimalField('nonnegative', {
    empty: 'Escriba el costo unitario',
    belowMin: 'El costo no puede ser negativo',
  }),
});

type FormInput = z.input<typeof schema>;
type FormValues = z.output<typeof schema>;

// Error codes of the API (api/openapi.yaml) that belong to one field.
const fieldByErrorCode: Record<string, keyof FormInput> = {
  product_not_found: 'product',
  location_not_found: 'location_id',
  location_inactive: 'location_id',
  invalid_quantity: 'quantity',
  invalid_unit_cost: 'unit_cost',
};

/**
 * /inventario/entrada: goods that arrive at a location. ?producto= and ?sede=
 * come chosen from the kardex or the stock.
 */
export function ReceiptPage() {
  const [params] = useSearchParams();
  const locations = useLocations();
  const preset = useProduct(params.get('producto') ?? undefined);

  const title = 'Registrar entrada';
  const description = 'La mercancía que llega suma a la existencia de la sede y actualiza el costo promedio.';

  if (locations.isPending || (params.has('producto') && preset.isPending)) {
    return (
      <Page title={title} description={description}>
        <Spinner label="Cargando…" className={styles.loading} />
      </Page>
    );
  }
  if (locations.isError) {
    return (
      <Page title={title} description={description}>
        <Alert tone="danger" title="No pudimos cargar las sedes">
          {locations.error.message}
        </Alert>
      </Page>
    );
  }
  const active = locations.data.filter((location) => location.active);
  if (active.length === 0) {
    return (
      <Page title={title} description={description}>
        <EmptyState
          icon={<IconBox />}
          title="Aún no hay sedes activas"
          description="Para registrar mercancía primero debe existir una sede o bodega."
          action={
            <ButtonLink to="/inventario" variant="secondary">
              Ir a inventario
            </ButtonLink>
          }
        />
      </Page>
    );
  }

  const presetLocation = active.find((location) => location.id === params.get('sede'));
  return (
    <Page title={title} description={description}>
      <ReceiptForm
        locations={active.map((location) => ({ value: location.id, label: location.name }))}
        initialProduct={preset.data ?? null}
        initialLocation={presetLocation?.id ?? (active.length === 1 ? active[0]?.id : undefined) ?? ''}
      />
    </Page>
  );
}

type FormProps = {
  locations: { value: string; label: string }[];
  initialProduct: Product | null;
  initialLocation: string;
};

function ReceiptForm({ locations, initialProduct, initialLocation }: FormProps) {
  const queryClient = useQueryClient();
  const { toast } = useToast();
  const [formError, setFormError] = useState<string | null>(null);
  const {
    register,
    control,
    handleSubmit,
    setError,
    setValue,
    setFocus,
    resetField,
    formState: { errors, isSubmitting },
  } = useForm<FormInput, unknown, FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      product: initialProduct,
      location_id: initialLocation,
      quantity: '',
      unit_cost: initialProduct?.cost ? toDecimalInput(initialProduct.cost) : '',
    },
  });
  const product = useWatch({ control, name: 'product' });
  const unit = product ? unitShort(product.base_unit) : undefined;

  // Coming from the kardex, the product is already there: the quantity is next.
  useEffect(() => {
    if (initialProduct) setFocus('quantity');
  }, [initialProduct, setFocus]);

  function choose(next: Product | null) {
    setValue('product', next, { shouldValidate: Boolean(errors.product) });
    // The last cost is a good start; only those who see costs (catalog:write) get it.
    if (next?.cost) setValue('unit_cost', toDecimalInput(next.cost));
  }

  async function receive(values: FormValues) {
    setFormError(null);
    if (!values.product) return;
    try {
      const movement = unwrap(
        await api.POST('/api/v1/inventory/receipts', {
          body: {
            product_id: values.product.id,
            location_id: values.location_id,
            quantity: values.quantity,
            unit_cost: values.unit_cost,
          },
        }),
      );
      void queryClient.invalidateQueries({ queryKey: inventoryKeys.all });
      const productUnit = unitShort(values.product.base_unit);
      const location = locations.find((l) => l.value === values.location_id)?.label ?? 'La sede';
      toast({
        tone: 'success',
        title: 'Entrada registrada',
        description: `${formatQuantity(movement.quantity)} ${productUnit} de ${values.product.name}. ${location} queda con ${formatQuantity(movement.balance_after)} ${productUnit}.`,
      });
      // Ready for the next product of the same delivery, in the same location.
      resetField('product', { defaultValue: null });
      resetField('quantity', { defaultValue: '' });
      resetField('unit_cost', { defaultValue: '' });
    } catch (error) {
      const { code, message } = describeError(error);
      const field = fieldByErrorCode[code ?? ''];
      if (field) setError(field, { message }, { shouldFocus: field !== 'product' && field !== 'location_id' });
      else setFormError(message);
    }
  }

  return (
    <form className={styles.form} noValidate onSubmit={handleSubmit(receive)}>
      {formError && <Alert tone="danger">{formError}</Alert>}
      <Controller
        control={control}
        name="product"
        render={({ field, fieldState }) => (
          <ProductPicker
            label="Producto"
            value={field.value ?? null}
            onChange={choose}
            onPicked={() => setFocus('quantity')}
            error={fieldState.error?.message}
          />
        )}
      />
      <Field label="Sede" error={errors.location_id?.message}>
        <Controller
          control={control}
          name="location_id"
          render={({ field }) => (
            <Select
              options={locations}
              value={field.value || undefined}
              onValueChange={field.onChange}
              placeholder="Elija la sede"
            />
          )}
        />
      </Field>
      <div className={styles.amounts}>
        <Field label="Cantidad" hint={unit ? `En ${unit}.` : 'En la unidad del producto.'} error={errors.quantity?.message}>
          <Input inputMode="decimal" align="end" autoComplete="off" {...register('quantity')} />
        </Field>
        <Field
          label="Costo unitario"
          hint={unit ? `Sin IVA, por ${unit}.` : 'Sin IVA, por unidad.'}
          error={errors.unit_cost?.message}
        >
          <Input inputMode="decimal" align="end" autoComplete="off" {...register('unit_cost')} />
        </Field>
      </div>
      <div className={styles.footer}>
        <Button type="submit" size="lg" loading={isSubmitting}>
          Registrar entrada
        </Button>
      </div>
    </form>
  );
}
