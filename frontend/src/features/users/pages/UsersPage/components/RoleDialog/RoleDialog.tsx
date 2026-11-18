import { zodResolver } from '@hookform/resolvers/zod';
import { useForm } from 'react-hook-form';
import { FormDialog } from '@/shared/forms/FormDialog/FormDialog';
import { SelectController } from '@/shared/forms/SelectController/SelectController';
import { useSaveDialog } from '@/shared/forms/useSaveDialog';
import { Field } from '@/shared/ui';
import { useChangeRole } from '../../../../api/users';
import type { User } from '../../../../api/types';
import { roleSchema, type RoleValues } from '../../../../model/inviteSchema';
import type { RoleOption } from '../../hooks/useRoleOptions';

type Props = { user: User | null; roles: RoleOption[]; onClose: () => void };

// owner_only and last_owner are no field's: they stay as the form's message.
const fields = {};

/** Gives a user another role; it applies from their next request. */
export function RoleDialog({ user, roles, onClose }: Props) {
  const form = useForm({ resolver: zodResolver(roleSchema), values: { role: user?.role ?? 'cashier' } });
  const { mutateAsync } = useChangeRole();
  const { formError, onSubmit, close } = useSaveDialog(form, {
    fields,
    save: (body: RoleValues) => (user ? mutateAsync({ user, body }) : Promise.reject(new Error('no user to change'))),
    success: (saved) => ({ title: 'Rol cambiado', description: saved.name }),
    onClose,
  });

  return (
    <FormDialog
      open={user !== null}
      onClose={close}
      title={`Rol de ${user?.name ?? ''}`}
      description="El cambio aplica de inmediato, sin que la persona vuelva a ingresar."
      size="sm"
      onSubmit={onSubmit}
      isSubmitting={form.formState.isSubmitting}
      formError={formError}
    >
      <Field label="Rol">
        <SelectController control={form.control} name="role" options={roles} />
      </Field>
    </FormDialog>
  );
}
