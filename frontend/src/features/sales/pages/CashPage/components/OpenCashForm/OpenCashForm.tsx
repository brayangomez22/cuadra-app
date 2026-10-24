import { Controller } from 'react-hook-form';
import { Alert, Button, Field, Input, Select } from '@/shared/ui';
import { useOpenCashForm } from '../../hooks/useOpenCashForm';
import styles from './OpenCashForm.module.scss';

/** Opens the drawer: the location, when there are several, and the float counted. */
export function OpenCashForm({ locations }: { locations: { value: string; label: string }[] }) {
  const { form, formError, submit, isPending } = useOpenCashForm(locations);
  const { errors } = form.formState;

  return (
    <form className={styles.root} noValidate onSubmit={(event) => void submit(event)}>
      {formError && <Alert tone="danger">{formError}</Alert>}
      {locations.length > 1 && (
        <Field label="Sede" error={errors.location_id?.message}>
          <Controller
            control={form.control}
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
      )}
      {locations.length === 1 && errors.location_id && <Alert tone="danger">{errors.location_id.message}</Alert>}
      <Field
        label="Base inicial"
        hint="El efectivo que hay en el cajón al empezar."
        error={errors.opening_float?.message}
      >
        <Input inputMode="decimal" align="end" autoComplete="off" autoFocus {...form.register('opening_float')} />
      </Field>
      <div className={styles.footer}>
        <Button type="submit" size="lg" loading={isPending}>
          Abrir caja
        </Button>
      </div>
    </form>
  );
}
