// The alternate units a variant is sold and bought in (docs/decisiones.md,
// T26): each one is saved on its own with PUT or DELETE …/units/{unitId}, not
// with the product's form, because it belongs to a variant that must exist.
import { zodResolver } from '@hookform/resolvers/zod';
import { useQueryClient } from '@tanstack/react-query';
import { useId, useState } from 'react';
import { Controller, useForm, useWatch } from 'react-hook-form';
import { z } from 'zod';
import { api } from '@/shared/api/client';
import { describeError, unwrap } from '@/shared/api/errors';
import { decimalField } from '@/shared/forms/decimalField';
import { formatMoney, formatQuantity, parseDecimalInput, toDecimalInput } from '@/shared/format';
import { toUnits } from '@/shared/money/decimal';
import {
  Alert,
  Button,
  Dialog,
  DialogClose,
  Field,
  IconButton,
  IconEdit,
  IconPlus,
  IconTrash,
  Input,
  Select,
  useToast,
} from '@/shared/ui';
import { catalogKeys, type Product, type ProductVariant, type UnitOfMeasure, type VariantUnit } from './api';
import { hasAxes, suggestedUnitPrice, variantName } from './variants';
import styles from './VariantUnits.module.scss';

type Props = {
  /** The saved product; a new one has no variants to add units to yet. */
  product?: Product;
  /** The tenant's units. */
  units: UnitOfMeasure[];
};

/** The "Unidades de venta" of a product: for each variant, its alternate units. */
export function VariantUnits({ product, units }: Props) {
  const queryClient = useQueryClient();
  const { toast } = useToast();
  const [editing, setEditing] = useState<{ variant: ProductVariant; unit?: VariantUnit } | null>(null);

  if (!product) {
    return <p className={styles.hint}>Guarde el producto para agregar las otras unidades en que se vende.</p>;
  }
  const base = product.unit.code;
  const byAxes = hasAxes(product);

  async function remove(variant: ProductVariant, unit: VariantUnit) {
    if (!product) return;
    try {
      const saved = unwrap(
        await api.DELETE('/api/v1/products/{productId}/variants/{variantId}/units/{unitId}', {
          params: { path: { productId: product.id, variantId: variant.id, unitId: unit.unit.id } },
        }),
      );
      updated(queryClient, saved);
      toast({ tone: 'success', title: 'Unidad quitada', description: unit.unit.name });
    } catch (error) {
      toast({ tone: 'danger', title: 'No se pudo quitar la unidad', description: describeError(error).message });
    }
  }

  return (
    <>
      <p className={styles.hint}>
        Otras presentaciones en que se vende y se compra, cada una con su precio. El inventario se lleva en {base}.
      </p>
      <div className={styles.root}>
        {product.variants.map((variant) => {
          const label = byAxes ? variant.label : '';
          return (
            <div key={variant.id} className={styles.variant} role={byAxes ? 'group' : undefined} aria-label={label || undefined}>
              {byAxes && <span className={styles.variantLabel}>{label}</span>}
              {variant.units.length === 0 ? (
                <p className={styles.hint}>Sin unidades de venta: se vende solo por {base}.</p>
              ) : (
                <ul className={styles.list}>
                  {variant.units.map((unit) => (
                    <li key={unit.unit.id} className={styles.unit} aria-label={unit.unit.code}>
                      <div className={styles.unitText}>
                        <span className={styles.equivalence}>
                          1 {unit.unit.code} = {formatQuantity(unit.factor)} {base}
                        </span>
                        <span className={styles.meta}>
                          {formatMoney(unit.price)} sin IVA · {formatMoney(unit.price_with_tax)} con IVA
                        </span>
                        {unit.barcode && <span className={styles.barcode}>{unit.barcode}</span>}
                      </div>
                      <div className={styles.actions}>
                        <IconButton
                          label={`Editar ${unit.unit.code}`}
                          icon={<IconEdit />}
                          onClick={() => setEditing({ variant, unit })}
                        />
                        <IconButton
                          label={`Quitar ${unit.unit.code}`}
                          icon={<IconTrash />}
                          variant="danger"
                          onClick={() => void remove(variant, unit)}
                        />
                      </div>
                    </li>
                  ))}
                </ul>
              )}
              <Button
                variant="secondary"
                icon={<IconPlus />}
                className={styles.add}
                aria-label={byAxes ? `Agregar unidad a ${label}` : undefined}
                onClick={() => setEditing({ variant })}
              >
                Agregar unidad
              </Button>
            </div>
          );
        })}
      </div>
      {editing && (
        <UnitDialog
          key={`${editing.variant.id}-${editing.unit?.unit.id ?? 'new'}`}
          product={product}
          variant={editing.variant}
          unit={editing.unit}
          units={units}
          onClose={() => setEditing(null)}
        />
      )}
    </>
  );
}

// The product the API answers replaces the cached one, so the screens that
// sell it see its units.
function updated(queryClient: ReturnType<typeof useQueryClient>, product: Product) {
  queryClient.setQueryData(catalogKeys.product(product.id), product);
  void queryClient.invalidateQueries({ queryKey: catalogKeys.lists });
}

const schema = z.object({
  unitId: z.string().min(1, 'Elija la unidad'),
  factor: decimalField('positive', {
    empty: 'Escriba cuántas trae',
    belowMin: 'Debe traer más de cero',
  }),
  price: decimalField('positive', {
    empty: 'Escriba el precio',
    belowMin: 'El precio debe ser mayor que cero',
  }),
  barcode: z.string().trim().max(64, 'El código de barras puede tener máximo 64 caracteres'),
});

type FormInput = z.input<typeof schema>;
type FormValues = z.output<typeof schema>;

// Error codes of PUT …/units/{unitId} (api/openapi.yaml) that belong to one field.
const fieldByErrorCode: Record<string, keyof FormInput> = {
  unit_not_found: 'unitId',
  alternate_unit_is_base: 'unitId',
  invalid_unit_factor: 'factor',
  invalid_price: 'price',
  invalid_barcode: 'barcode',
  barcode_taken: 'barcode',
};

type DialogProps = {
  product: Product;
  variant: ProductVariant;
  /** The unit to edit; without it, the dialog adds one. */
  unit?: VariantUnit;
  units: UnitOfMeasure[];
  onClose: () => void;
};

function UnitDialog({ product, variant, unit, units, onClose }: DialogProps) {
  const formId = useId();
  const queryClient = useQueryClient();
  const { toast } = useToast();
  const [formError, setFormError] = useState<string | null>(null);
  const base = product.unit.code;
  const {
    register,
    control,
    handleSubmit,
    setError,
    setValue,
    formState: { errors, isSubmitting },
  } = useForm<FormInput, unknown, FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      unitId: unit?.unit.id ?? '',
      factor: unit ? toDecimalInput(unit.factor) : '',
      price: unit ? toDecimalInput(unit.price) : '',
      barcode: unit?.barcode ?? '',
    },
  });
  const factor = readFactor(useWatch({ control, name: 'factor' }));
  const suggested = factor && suggestedUnitPrice(variant.price, factor);

  // The base unit is not an alternate one, and a unit the variant has is edited, not added again.
  const taken = new Set(variant.units.map((u) => u.unit.id));
  const options = units
    .filter((u) => u.id !== product.base_unit_id && (u.id === unit?.unit.id || !taken.has(u.id)))
    .map((u) => ({ value: u.id, label: `${u.name} (${u.code})` }));

  async function save(values: FormValues) {
    setFormError(null);
    try {
      const saved = unwrap(
        await api.PUT('/api/v1/products/{productId}/variants/{variantId}/units/{unitId}', {
          params: { path: { productId: product.id, variantId: variant.id, unitId: values.unitId } },
          body: { factor: values.factor, price: values.price, ...(values.barcode && { barcode: values.barcode }) },
        }),
      );
      updated(queryClient, saved);
      const name = units.find((u) => u.id === values.unitId)?.name ?? '';
      toast({ tone: 'success', title: 'Unidad guardada', description: name });
      onClose();
    } catch (error) {
      const { code, message } = describeError(error);
      const field = fieldByErrorCode[code ?? ''];
      if (field) setError(field, { message }, { shouldFocus: true });
      else setFormError(message);
    }
  }

  return (
    <Dialog
      open
      onOpenChange={(open) => !open && onClose()}
      title={unit ? 'Editar unidad de venta' : 'Agregar unidad de venta'}
      description={variantName(product.name, hasAxes(product) ? variant.label : '')}
      footer={
        <>
          <DialogClose asChild>
            <Button variant="ghost">Cancelar</Button>
          </DialogClose>
          <Button type="submit" form={formId} loading={isSubmitting}>
            Guardar unidad
          </Button>
        </>
      }
    >
      <form
        id={formId}
        className={styles.form}
        noValidate
        // The dialog renders in a portal, but React bubbles its submit to the
        // product's form around this section: it must stop here.
        onSubmit={(event) => {
          event.stopPropagation();
          void handleSubmit(save)(event);
        }}
      >
        {formError && <Alert tone="danger">{formError}</Alert>}
        <Field label="Unidad de venta" error={errors.unitId?.message}>
          <Controller
            control={control}
            name="unitId"
            render={({ field }) => (
              <Select
                options={options}
                value={field.value || undefined}
                onValueChange={field.onChange}
                placeholder="Elija la unidad"
                disabled={unit !== undefined}
              />
            )}
          />
        </Field>
        <Field label={`Cuántas ${base} trae cada una`} hint="Puede tener decimales: 0,5 o 2,5." error={errors.factor?.message}>
          <Input inputMode="decimal" align="end" autoComplete="off" {...register('factor')} />
        </Field>
        <Field
          label="Precio (sin IVA)"
          hint={
            factor &&
            suggested &&
            `Sugerido: ${formatMoney(suggested)} (${formatQuantity(factor)} × ${formatMoney(variant.price)}).`
          }
          error={errors.price?.message}
        >
          <Input inputMode="decimal" align="end" autoComplete="off" {...register('price')} />
        </Field>
        {suggested && (
          <Button
            variant="ghost"
            className={styles.suggest}
            onClick={() => setValue('price', toDecimalInput(suggested), { shouldValidate: true })}
          >
            Usar el sugerido
          </Button>
        )}
        <Field label="Código de barras" optional error={errors.barcode?.message}>
          <Input autoComplete="off" spellCheck={false} {...register('barcode')} />
        </Field>
      </form>
    </Dialog>
  );
}

/** The factor typed, as the API's decimal, when it is a positive one. */
function readFactor(text: string): string | undefined {
  const parsed = parseDecimalInput(text);
  return parsed.ok && toUnits(parsed.value) > 0n ? parsed.value : undefined;
}
