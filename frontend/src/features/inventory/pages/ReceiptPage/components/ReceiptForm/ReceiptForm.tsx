import { FormProvider } from 'react-hook-form';
import { useModule } from '@/features/auth';
import { SelectController } from '@/shared/forms/SelectController/SelectController';
import { useFieldError } from '@/shared/forms/useFieldError';
import { Alert, Button, Field } from '@/shared/ui';
import { useReceiptForm, type ReceiptSetup } from '../../hooks/useReceiptForm';
import { AmountFields } from '../AmountFields/AmountFields';
import { ProductField } from '../ProductField/ProductField';
import { UnitField } from '../../../../components/UnitField/UnitField';
import styles from './ReceiptForm.module.scss';

/** The product, the location, the unit when the variant has others, the quantity and the cost. */
export function ReceiptForm(setup: ReceiptSetup) {
  const alternateUnits = useModule('unit_conversions');
  const { form, formError, choose, chooseUnit, submit, isPending } = useReceiptForm(setup);
  const errorOf = useFieldError(form.control, 'location_id');

  return (
    <FormProvider {...form}>
      <form className={styles.root} noValidate onSubmit={(event) => void submit(event)}>
        {formError && <Alert tone="danger">{formError}</Alert>}
        <ProductField onChange={choose} />
        <Field label="Sede" error={errorOf('location_id')}>
          <SelectController
            control={form.control}
            name="location_id"
            options={setup.locations}
            placeholder="Elija la sede"
          />
        </Field>
        {alternateUnits && <UnitField hint="En la que llega la mercancía." onChange={chooseUnit} />}
        <AmountFields />
        <div className={styles.footer}>
          <Button type="submit" size="lg" loading={isPending}>
            Registrar entrada
          </Button>
        </div>
      </form>
    </FormProvider>
  );
}
