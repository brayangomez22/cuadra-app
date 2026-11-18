import { zodResolver } from '@hookform/resolvers/zod';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { FormDialog } from '@/shared/forms/FormDialog/FormDialog';
import { useSaveDialog } from '@/shared/forms/useSaveDialog';
import { Button, Field, PasswordInput } from '@/shared/ui';
import { useChangePassword } from '../../api/password';
import { changePasswordSchema, type ChangePasswordValues } from '../../model/passwordSchemas';

const empty = { current_password: '', new_password: '' };

// The field each 422 code of POST /me/password belongs to.
const fields = {
  wrong_password: 'current_password',
  password_too_short: 'new_password',
  password_too_long: 'new_password',
  password_too_common: 'new_password',
} as const;

/** "Cambiar contraseña" in the account panel: the user's own password, with the current one. */
export function ChangePasswordDialog({ className }: { className?: string }) {
  const [open, setOpen] = useState(false);
  const form = useForm({ resolver: zodResolver(changePasswordSchema), defaultValues: empty });
  const { mutateAsync } = useChangePassword();
  const { formError, onSubmit, close } = useSaveDialog(form, {
    fields,
    save: (values: ChangePasswordValues) => mutateAsync(values),
    success: () => ({ title: 'Contraseña cambiada' }),
    onClose: () => {
      form.reset(empty);
      setOpen(false);
    },
  });
  const { errors, isSubmitting } = form.formState;

  return (
    <>
      <Button variant="ghost" className={className} onClick={() => setOpen(true)}>
        Cambiar contraseña
      </Button>
      <FormDialog
        open={open}
        onClose={close}
        title="Cambiar contraseña"
        description="Sus sesiones abiertas siguen abiertas."
        size="sm"
        onSubmit={onSubmit}
        isSubmitting={isSubmitting}
        formError={formError}
      >
        <Field label="Contraseña actual" error={errors.current_password?.message}>
          <PasswordInput autoComplete="current-password" {...form.register('current_password')} />
        </Field>
        <Field label="Contraseña nueva" hint="Al menos 8 caracteres." error={errors.new_password?.message}>
          <PasswordInput autoComplete="new-password" {...form.register('new_password')} />
        </Field>
      </FormDialog>
    </>
  );
}
