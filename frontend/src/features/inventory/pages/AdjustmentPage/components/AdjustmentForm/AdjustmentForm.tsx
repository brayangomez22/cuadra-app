import { FormProvider, useWatch } from 'react-hook-form';
import { SelectController } from '@/shared/forms/SelectController/SelectController';
import { useFieldError } from '@/shared/forms/useFieldError';
import { Alert, Button, Field } from '@/shared/ui';
import { BalanceChange } from '../../../../components/BalanceChange/BalanceChange';
import { ProductField } from '../../../../components/ProductField/ProductField';
import { QuantityField } from '../../../../components/QuantityField/QuantityField';
import { DIRECTIONS } from '../../../../model/stockChange/adjustmentSchema';
import { adjustmentDelta } from '../../../../model/stockChange/stockChange';
import { useAdjustmentForm, type AdjustmentSetup } from '../../hooks/useAdjustmentForm';
import { ReasonField } from '../ReasonField/ReasonField';
import styles from './AdjustmentForm.module.scss';

/** The product, the location, whether it leaves or enters, how much and why, with what it would leave. */
export function AdjustmentForm(setup: AdjustmentSetup) {
  const { form, formError, submit, isPending } = useAdjustmentForm(setup);
  const { control } = form;
  const errorOf = useFieldError(control);
  const [item, locationId, direction, quantity] = useWatch({
    control,
    name: ['item', 'location_id', 'direction', 'quantity'],
  });
  const delta = adjustmentDelta(direction, quantity);

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
        <Field label="Movimiento" error={errorOf('direction')}>
          <SelectController control={control} name="direction" options={DIRECTIONS} placeholder="¿Sale o entra?" />
        </Field>
        <QuantityField />
        {item && locationId && delta && (
          <BalanceChange
            variantId={item.variant.id}
            locationId={locationId}
            locationName={setup.locations.find((l) => l.value === locationId)?.label ?? 'La sede'}
            unit={item.product.unit.code}
            delta={delta}
          />
        )}
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
