import { zodResolver } from '@hookform/resolvers/zod';
import { useForm } from 'react-hook-form';
import { useApiFormErrors } from '@/shared/forms/useApiFormErrors';
import { Alert, Button, Field, PasswordInput, Stack } from '@/shared/ui';
import { useResetPassword } from '../../../../api/links';
import { resetPasswordSchema, type ResetPasswordValues } from '../../../../model/passwordSchemas';

type Props = { token: string; onDone: () => void };

// The 422 codes of POST /auth/password-reset are the password's; a used or
// expired link (410) stays as the form's message.
const fields = {
  password_too_short: 'password',
  password_too_long: 'password',
  password_too_common: 'password',
} as const;

export function ResetPasswordForm({ token, onDone }: Props) {
  const form = useForm({ resolver: zodResolver(resetPasswordSchema), defaultValues: { password: '' } });
  const { formError, handleError, clearFormError } = useApiFormErrors(form.setError, fields);
  const { mutateAsync } = useResetPassword();
  const { errors, isSubmitting } = form.formState;

  async function submit({ password }: ResetPasswordValues) {
    clearFormError();
    try {
      await mutateAsync({ token, password });
      onDone();
    } catch (error) {
      handleError(error);
    }
  }

  return (
    <form noValidate onSubmit={form.handleSubmit(submit)}>
      <Stack gap="4">
        {formError && <Alert tone="danger">{formError}</Alert>}
        <Field label="Contraseña nueva" hint="Al menos 8 caracteres." error={errors.password?.message}>
          <PasswordInput autoComplete="new-password" autoFocus {...form.register('password')} />
        </Field>
        <Button type="submit" size="lg" loading={isSubmitting}>
          Guardar contraseña
        </Button>
      </Stack>
    </form>
  );
}
