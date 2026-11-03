import { FormProvider, useWatch } from 'react-hook-form';
import { SelectController } from '@/shared/forms/SelectController/SelectController';
import { useFieldError } from '@/shared/forms/useFieldError';
import { Alert, Button, Field, SegmentedControl } from '@/shared/ui';
import { BalanceChange } from '../../../../components/BalanceChange/BalanceChange';
import { ProductField } from '../../../../components/ProductField/ProductField';
import { QuantityField } from '../../../../components/QuantityField/QuantityField';
import { MODES } from '../../../../model/stockChange/adjustmentSchema';
import { adjustmentDelta } from '../../../../model/stockChange/stockChange';
import { useAdjustmentForm, type AdjustmentSetup } from '../../hooks/useAdjustmentForm';
import { ReasonField } from '../ReasonField/ReasonField';
import styles from './AdjustmentForm.module.scss';

/**
 * The product, the location, whether it leaves, enters or what there is now,
 * how much and why. What there is shows as soon as the product and the
 * location are chosen, and then what the adjustment would leave.
 */
export function AdjustmentForm(setup: AdjustmentSetup) {
  const { form, formError, submit, current, isPending } = useAdjustmentForm(setup);
  const { control, register } = form;
  const errorOf = useFieldError(control);
  const [item, locationId, direction, quantity] = useWatch({
    control,
    name: ['item', 'location_id', 'direction', 'quantity'],
  });
  const delta = adjustmentDelta(direction, quantity, current);

  return (
    <FormProvider {...form}>
      <form className={styles.root} noValidate onSubmit={(event) => void submit(event)}>
        {formError && <Alert tone="danger">{formError}</Alert>}
        <ProductField />
        <Field label="Sede" error={errorOf('location_id')}>
          <SelectController
            control={control}
            name="location_id"
            options={setup.locations}
            placeholder="Elija la sede"
          />
        </Field>
        {item && locationId && (
          <BalanceChange
            variantId={item.variant.id}
            locationId={locationId}
            locationName={setup.locations.find((l) => l.value === locationId)?.label ?? 'La sede'}
            unit={item.product.unit.code}
            delta={delta}
          />
        )}
        <SegmentedControl legend="Movimiento" options={MODES} error={errorOf('direction')} {...register('direction')} />
        <QuantityField label={direction === 'count' ? 'Cuántas hay' : 'Cantidad'} />
        <ReasonField />
        <div className={styles.footer}>
          <Button type="submit" size="lg" loading={isPending}>
            Registrar ajuste
          </Button>
        </div>
      </form>
    </FormProvider>
  );
}
