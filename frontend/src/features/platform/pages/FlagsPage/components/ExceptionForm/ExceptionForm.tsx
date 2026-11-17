import { zodResolver } from '@hookform/resolvers/zod';
import { useForm } from 'react-hook-form';
import { useApiFormErrors } from '@/shared/forms/useApiFormErrors';
import { Alert, Button, Field, Input, SegmentedControl } from '@/shared/ui';
import { useSetFlagException } from '../../../../api/flags';
import { exceptionDefaults, exceptionSchema } from '../../../../model/exceptionSchema';
import styles from './ExceptionForm.module.scss';

const values = [
  { value: 'on', label: 'Prendida' },
  { value: 'off', label: 'Apagada' },
];

/** Gives one business its own value of a flag, by the business's ID. */
export function ExceptionForm({ flagKey }: { flagKey: string }) {
  const form = useForm({ resolver: zodResolver(exceptionSchema), defaultValues: exceptionDefaults });
  const { mutateAsync } = useSetFlagException();
  const { formError, handleError } = useApiFormErrors(form.setError, { unknown_tenant: 'tenant_id' });
  const submit = form.handleSubmit(async ({ tenant_id, value }) => {
    try {
      await mutateAsync({ key: flagKey, tenantId: tenant_id, enabled: value === 'on' });
      form.reset(exceptionDefaults);
    } catch (error) {
      handleError(error);
    }
  });
  return (
    <form className={styles.root} noValidate onSubmit={(event) => void submit(event)}>
      {formError && <Alert tone="danger">{formError}</Alert>}
      <Field label="ID de la empresa" error={form.formState.errors.tenant_id?.message}>
        <Input autoComplete="off" spellCheck={false} {...form.register('tenant_id')} />
      </Field>
      <SegmentedControl legend="Valor para esa empresa" options={values} {...form.register('value')} />
      <Button type="submit" variant="secondary" loading={form.formState.isSubmitting}>
        Agregar excepción
      </Button>
    </form>
  );
}
