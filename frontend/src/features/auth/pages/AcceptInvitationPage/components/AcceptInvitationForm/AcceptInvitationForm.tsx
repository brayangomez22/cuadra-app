import { zodResolver } from '@hookform/resolvers/zod';
import { useForm } from 'react-hook-form';
import { Alert, Button, Field, Input, PasswordInput, Stack } from '@/shared/ui';
import { acceptInvitationSchema } from '../../../../model/passwordSchemas';
import { useAcceptInvitation } from '../../hooks/useAcceptInvitation';

type Props = {
  token: string;
  /** The name the inviter wrote, which the invitee may change. */
  defaultName: string;
};

export function AcceptInvitationForm({ token, defaultName }: Props) {
  const form = useForm({
    resolver: zodResolver(acceptInvitationSchema),
    defaultValues: { name: defaultName, password: '' },
  });
  const { formError, submit } = useAcceptInvitation(token, form.setError);
  const { errors, isSubmitting } = form.formState;

  return (
    <form noValidate onSubmit={form.handleSubmit(submit)}>
      <Stack gap="4">
        {formError && <Alert tone="danger">{formError}</Alert>}
        <Field label="Su nombre" error={errors.name?.message}>
          <Input autoComplete="name" {...form.register('name')} />
        </Field>
        <Field
          label="Contraseña"
          hint="Al menos 8 caracteres. Una frase que recuerde es más segura que una palabra con números."
          error={errors.password?.message}
        >
          <PasswordInput autoComplete="new-password" autoFocus {...form.register('password')} />
        </Field>
        <Button type="submit" size="lg" loading={isSubmitting}>
          Entrar a la empresa
        </Button>
      </Stack>
    </form>
  );
}
