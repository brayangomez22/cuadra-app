import { zodResolver } from '@hookform/resolvers/zod';
import { useForm } from 'react-hook-form';
import { FormDialog } from '@/shared/forms/FormDialog/FormDialog';
import { SelectController } from '@/shared/forms/SelectController/SelectController';
import { useSaveDialog } from '@/shared/forms/useSaveDialog';
import { Field, Input } from '@/shared/ui';
import { useInvite } from '../../../../api/invitations';
import { inviteSchema, type InviteValues } from '../../../../model/inviteSchema';
import type { RoleOption } from '../../hooks/useRoleOptions';

type Created = Awaited<ReturnType<ReturnType<typeof useInvite>['mutateAsync']>>;

type Props = {
  open: boolean;
  roles: RoleOption[];
  onClose: () => void;
  /** The invitation was created: the page shows its link. */
  onInvited: (created: Created) => void;
};

const empty = { email: '', name: '' } as unknown as InviteValues;
const fields = { email_taken: 'email', invalid_email: 'email', invalid_user_name: 'name', owner_only: 'role' } as const;

/** Invites someone with a role; the link to share comes next. */
export function InviteDialog({ open, roles, onClose, onInvited }: Props) {
  const form = useForm({ resolver: zodResolver(inviteSchema), defaultValues: empty });
  const { mutateAsync } = useInvite();
  const { formError, onSubmit, close } = useSaveDialog(form, {
    fields,
    save: (values: InviteValues) => mutateAsync(values),
    success: (created) => {
      onInvited(created);
      return { title: 'Invitación creada', description: created.invitation.name };
    },
    onClose: () => {
      form.reset(empty);
      onClose();
    },
  });
  const { errors, isSubmitting } = form.formState;

  return (
    <FormDialog
      open={open}
      onClose={close}
      title="Invitar usuario"
      description="Genera un enlace para que la persona elija su contraseña y entre con el rol que le dé."
      onSubmit={onSubmit}
      isSubmitting={isSubmitting}
      formError={formError}
    >
      <Field label="Correo" hint="Con él ingresará." error={errors.email?.message}>
        <Input type="email" inputMode="email" autoComplete="off" {...form.register('email')} />
      </Field>
      <Field label="Nombre" error={errors.name?.message}>
        <Input autoComplete="off" {...form.register('name')} />
      </Field>
      <Field label="Rol" error={errors.role?.message}>
        <SelectController control={form.control} name="role" options={roles} placeholder="Elija el rol" />
      </Field>
    </FormDialog>
  );
}
