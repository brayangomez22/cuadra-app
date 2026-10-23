import { zodResolver } from '@hookform/resolvers/zod';
import { useQueryClient } from '@tanstack/react-query';
import { useEffect, useState } from 'react';
import { Controller, useForm, useWatch } from 'react-hook-form';
import { useSearchParams } from 'react-router';
import { z } from 'zod';
import { useAuth } from '@/features/auth';
import {
  defaultVariant,
  ProductPicker,
  suggestedUnitPrice,
  useProduct,
  variantName,
  type Product,
  type Sellable,
} from '@/features/catalog';
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
import { paths } from '@/app/routes/paths';

// Same limits as ReceiptInput in api/openapi.yaml.
const schema = z.object({
  item: z.custom<Sellable | null>().refine((item) => item !== null, 'Elija el producto'),
  location_id: z.string().min(1, 'Elija la sede'),
  quantity: decimalField('positive', {
    empty: 'Escriba la cantidad',
    belowMin: 'La cantidad debe ser mayor que cero',
  }),
  unit_cost: decimalField('nonnegative', {
    empty: 'Escriba el costo unitario',
    belowMin: 'El costo no puede ser negativo',
  }),
  /** An alternate unit of the variant (docs/decisiones.md, T26); empty is the product's unit. */
  unit_id: z.string(),
});

type FormInput = z.input<typeof schema>;
type FormValues = z.output<typeof schema>;

// Error codes of the API (api/openapi.yaml) that belong to one field.
const fieldByErrorCode: Record<string, keyof FormInput> = {
  product_not_found: 'item',
  location_not_found: 'location_id',
  location_inactive: 'location_id',
  invalid_quantity: 'quantity',
  invalid_unit_cost: 'unit_cost',
  unit_not_available: 'unit_id',
  quantity_precision: 'quantity',
};

// The Select's value for the product's unit: a unit id is never empty.
const BASE_UNIT = 'base';

/**
 * /inventario/entrada: goods that arrive at a location. ?producto= (with
 * ?variante=, or its default variant) and ?sede= come chosen from the kardex
 * or the stock.
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
            <ButtonLink to={paths.inventory.stock()} variant="secondary">
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
        initialItem={preset.data ? presetItem(preset.data, params.get('variante')) : null}
        initialLocation={presetLocation?.id ?? (active.length === 1 ? active[0]?.id : undefined) ?? ''}
      />
    </Page>
  );
}

function presetItem(product: Product, variantId: string | null): Sellable {
  return { product, variant: product.variants.find((v) => v.id === variantId) ?? defaultVariant(product) };
}

type FormProps = {
  locations: { value: string; label: string }[];
  initialItem: Sellable | null;
  initialLocation: string;
};

function ReceiptForm({ locations, initialItem, initialLocation }: FormProps) {
  const auth = useAuth();
  const alternateUnits = auth.status === 'authenticated' && auth.user.modules.includes('unit_conversions');
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
      item: initialItem,
      location_id: initialLocation,
      quantity: '',
      unit_cost: initialItem?.variant.cost ? toDecimalInput(initialItem.variant.cost) : '',
      unit_id: '',
    },
  });
  const item = useWatch({ control, name: 'item' });
  const unitId = useWatch({ control, name: 'unit_id' });
  const base = item?.product.unit.code;
  const alternate = item?.variant.units.find((u) => u.unit.id === unitId);
  const unit = alternate ? alternate.unit.code : base;
  const quantityHint = alternate
    ? `En ${alternate.unit.code} de ${formatQuantity(alternate.factor)} ${base}.`
    : base
      ? `En ${base}.`
      : 'En la unidad del producto.';

  // Coming from the kardex, the product is already there: the quantity is next.
  useEffect(() => {
    if (initialItem) setFocus('quantity');
  }, [initialItem, setFocus]);

  function choose(next: Sellable | null) {
    setValue('item', next, { shouldValidate: Boolean(errors.item) });
    // A unit's barcode brings that unit, with the module on or off: the API
    // takes the units already defined.
    chooseUnit(next, next?.unit?.unit.id ?? '');
  }

  // The last cost is a good start, times the unit's factor; only those who
  // see costs (catalog:write) get it.
  function chooseUnit(sellable: Sellable | null | undefined, id: string) {
    setValue('unit_id', id);
    const cost = sellable?.variant.cost;
    if (!cost) return;
    const factor = sellable.variant.units.find((u) => u.unit.id === id)?.factor;
    setValue('unit_cost', toDecimalInput(factor ? suggestedUnitPrice(cost, factor) : cost));
  }

  async function receive(values: FormValues) {
    setFormError(null);
    if (!values.item) return;
    const { product, variant } = values.item;
    try {
      const movement = unwrap(
        await api.POST('/api/v1/inventory/receipts', {
          body: {
            variant_id: variant.id,
            location_id: values.location_id,
            quantity: values.quantity,
            unit_cost: values.unit_cost,
            ...(values.unit_id && { unit_id: values.unit_id }),
          },
        }),
      );
      void queryClient.invalidateQueries({ queryKey: inventoryKeys.all });
      const productUnit = product.unit.code;
      const received = variant.units.find((u) => u.unit.id === values.unit_id);
      // The movement is in the base unit: an entry in boxes says both.
      const quantity = received
        ? `${formatQuantity(values.quantity)} ${received.unit.code} (${formatQuantity(movement.quantity)} ${productUnit})`
        : `${formatQuantity(movement.quantity)} ${productUnit}`;
      const location = locations.find((l) => l.value === values.location_id)?.label ?? 'La sede';
      toast({
        tone: 'success',
        title: 'Entrada registrada',
        description: `${quantity} de ${variantName(product.name, variant.label)}. ${location} queda con ${formatQuantity(movement.balance_after)} ${productUnit}.`,
      });
      // Ready for the next product of the same delivery, in the same location.
      resetField('item', { defaultValue: null });
      resetField('quantity', { defaultValue: '' });
      resetField('unit_cost', { defaultValue: '' });
      resetField('unit_id', { defaultValue: '' });
    } catch (error) {
      const { code, message } = describeError(error);
      const field = fieldByErrorCode[code ?? ''];
      if (field) setError(field, { message }, { shouldFocus: field !== 'item' && field !== 'location_id' });
      else setFormError(message);
    }
  }

  return (
    <form className={styles.form} noValidate onSubmit={handleSubmit(receive)}>
      {formError && <Alert tone="danger">{formError}</Alert>}
      <Controller
        control={control}
        name="item"
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
      {alternateUnits && item && item.variant.units.length > 0 && (
        <Field label="Unidad" hint="En la que llega la mercancía." error={errors.unit_id?.message}>
          <Select
            options={[
              { value: BASE_UNIT, label: item.product.unit.code },
              ...item.variant.units.map((u) => ({
                value: u.unit.id,
                label: `${u.unit.code} (${formatQuantity(u.factor)} ${item.product.unit.code})`,
              })),
            ]}
            value={unitId || BASE_UNIT}
            onValueChange={(value) => chooseUnit(item, value === BASE_UNIT ? '' : value)}
          />
        </Field>
      )}
      <div className={styles.amounts}>
        <Field label="Cantidad" hint={quantityHint} error={errors.quantity?.message}>
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
