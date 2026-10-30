import { Controller, FormProvider } from 'react-hook-form';
import { ProductPicker } from '@/features/catalog';
import { useFieldError } from '@/shared/forms/useFieldError';
import { Alert, Button, Field, Input } from '@/shared/ui';
import { UnitField } from '../../../../components/UnitField/UnitField';
import { useCountEntryForm } from '../../hooks/useCountEntryForm';
import { ScanToCount } from '../ScanToCount/ScanToCount';
import styles from './CountEntryForm.module.scss';

/** The product (searched or scanned), the unit when it has others and how much there is. */
export function CountEntryForm({ countId }: { countId: string }) {
  const { form, formError, choose, chooseUnit, submit, isPending } = useCountEntryForm(countId);
  const errorOf = useFieldError(form.control, 'quantity');
  const focusQuantity = () => requestAnimationFrame(() => form.setFocus('quantity'));

  return (
    <FormProvider {...form}>
      <form className={styles.root} noValidate onSubmit={(event) => void submit(event)}>
        {formError && <Alert tone="danger">{formError}</Alert>}
        <Controller
          control={form.control}
          name="item"
          render={({ field, fieldState }) => (
            <ProductPicker
              label="Producto"
              ref={field.ref}
              value={field.value ?? null}
              onChange={choose}
              onPicked={focusQuantity}
              error={fieldState.error?.message}
            />
          )}
        />
        <ScanToCount onFound={choose} onClose={focusQuantity} />
        <UnitField hint="En la que contó." onChange={chooseUnit} />
        <Field label="Cantidad contada" hint="Si no hay ninguno, escriba 0." error={errorOf('quantity')}>
          <Input align="end" inputMode="decimal" autoComplete="off" {...form.register('quantity')} />
        </Field>
        <Button type="submit" size="lg" loading={isPending} className={styles.submit}>
          Registrar
        </Button>
      </form>
    </FormProvider>
  );
}
