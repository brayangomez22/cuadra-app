import { FormProvider } from 'react-hook-form';
import { SelectController } from '@/shared/forms/SelectController/SelectController';
import { useFieldError } from '@/shared/forms/useFieldError';
import { Alert, Button, Field } from '@/shared/ui';
import { ProductField } from '../../../../components/ProductField/ProductField';
import { QuantityField } from '../../../../components/QuantityField/QuantityField';
import { useTransferForm, type TransferSetup } from '../../hooks/useTransferForm';
import { TransferPreview } from '../TransferPreview/TransferPreview';
import styles from './TransferForm.module.scss';

/** The product, the two locations and how much, with what each location would have after it. */
export function TransferForm(setup: TransferSetup) {
  const { form, formError, submit, nameOf, isPending } = useTransferForm(setup);
  const { control } = form;
  const errorOf = useFieldError(control);

  return (
    <FormProvider {...form}>
      <form className={styles.root} noValidate onSubmit={(event) => void submit(event)}>
        {formError && <Alert tone="danger">{formError}</Alert>}
        <ProductField onPicked={() => form.setFocus('quantity')} />
        <Field label="Sede de origen" error={errorOf('from_location_id')}>
          <SelectController
            control={control}
            name="from_location_id"
            options={setup.locations}
            placeholder="Elija la sede"
          />
        </Field>
        <Field label="Sede de destino" error={errorOf('to_location_id')}>
          <SelectController
            control={control}
            name="to_location_id"
            options={setup.locations}
            placeholder="Elija la sede"
          />
        </Field>
        <QuantityField />
        <TransferPreview nameOf={nameOf} />
        <div className={styles.footer}>
          <Button type="submit" size="lg" loading={isPending}>
            Registrar traslado
          </Button>
        </div>
      </form>
    </FormProvider>
  );
}
