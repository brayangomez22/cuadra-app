import { zodResolver } from '@hookform/resolvers/zod';
import { useForm } from 'react-hook-form';
import type { UnitOfMeasure } from '@/features/catalog';
import { Field, Input } from '@/shared/ui';
import { useSaveUnit } from '../../../../api/units';
import { FormDialog } from '../../../../components/FormDialog/FormDialog';
import { useSaveDialog } from '../../../../hooks/useSaveDialog';
import { unitErrorFields, unitSchema } from '../../../../model/unitSchema';

type Props = {
  open: boolean;
  /** The unit to edit, or null to add one. */
  unit: UnitOfMeasure | null;
  onClose: () => void;
};

export function UnitDialog({ open, unit, onClose }: Props) {
  const form = useForm({
    resolver: zodResolver(unitSchema),
    values: { code: unit?.code ?? '', name: unit?.name ?? '' },
  });
  const { mutateAsync } = useSaveUnit();
  const { formError, onSubmit, close } = useSaveDialog(form, {
    fields: unitErrorFields,
    save: (body) => mutateAsync({ id: unit?.id, body }),
    success: (saved) => ({ title: 'Unidad guardada', description: saved.name }),
    onClose,
  });
  const { errors, isSubmitting } = form.formState;

  return (
    <FormDialog
      open={open}
      onClose={close}
      title={unit ? 'Editar unidad' : 'Agregar unidad'}
      description="El código es lo que se muestra junto a una cantidad: «2,5 m», «3 par»."
      onSubmit={onSubmit}
      isSubmitting={isSubmitting}
      formError={formError}
    >
      <Field label="Código" hint="Corto y sin espacios: und, m, kg, par" error={errors.code?.message}>
        <Input autoComplete="off" spellCheck={false} {...form.register('code')} />
      </Field>
      <Field label="Nombre" error={errors.name?.message}>
        <Input autoComplete="off" {...form.register('name')} />
      </Field>
    </FormDialog>
  );
}
