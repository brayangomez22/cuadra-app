import type { UseFormReturn } from 'react-hook-form';
import { Alert, Button, Field, Input, Stack } from '@/shared/ui';
import type { LoginValues } from '../../../../model/schemas';
import styles from './LoginForm.module.scss';

type Props = {
  form: UseFormReturn<LoginValues>;
  formError: string | null;
  onSubmit: (values: LoginValues) => Promise<void>;
};

export function LoginForm({ form, formError, onSubmit }: Props) {
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = form;

  return (
    <form noValidate onSubmit={handleSubmit(onSubmit)}>
      <Stack gap="4">
        {formError && <Alert tone="danger">{formError}</Alert>}
        <Field label="Correo" error={errors.email?.message}>
          <Input type="email" autoComplete="username" inputMode="email" autoFocus {...register('email')} />
        </Field>
        <Field label="Contraseña" error={errors.password?.message}>
          <Input type="password" autoComplete="current-password" {...register('password')} />
        </Field>
        <Button type="submit" size="lg" className={styles.submit} loading={isSubmitting}>
          Ingresar
        </Button>
      </Stack>
    </form>
  );
}
