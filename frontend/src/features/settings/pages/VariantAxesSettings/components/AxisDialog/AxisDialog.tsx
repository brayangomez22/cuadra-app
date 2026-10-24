import { zodResolver } from '@hookform/resolvers/zod';
import { useForm } from 'react-hook-form';
import type { VariantAxis } from '@/features/catalog';
import { Field, Input, Textarea } from '@/shared/ui';
import { useSaveVariantAxis } from '../../../../api/variantAxes';
import { FormDialog } from '../../../../components/FormDialog/FormDialog';
import { useSaveDialog } from '../../../../hooks/useSaveDialog';
import { axisErrorFields, axisSchema, type AxisValues } from '../../../../model/axisSchema';
import { joinLines, parseLines } from '../../../../model/lines';

// What the dialog says when it adds an axis and when it edits one.
const copy = {
  add: {
    title: 'Agregar eje',
    description: 'Las variantes eligen sus valores de esta lista, sin texto libre.',
    nameHint: 'Color, Tamaño, Presentación…',
  },
  edit: {
    title: 'Editar eje',
    description: 'El nombre no cambia. Un valor que alguna variante usa no se puede quitar.',
    nameHint: undefined,
  },
};

type Props = {
  open: boolean;
  /** The axis to edit, or null to add one. */
  axis: VariantAxis | null;
  onClose: () => void;
};

export function AxisDialog({ open, axis, onClose }: Props) {
  const form = useForm({
    resolver: zodResolver(axisSchema),
    values: axis ? { name: axis.name, values: joinLines(axis.values) } : { name: '', values: '' },
  });
  const { mutateAsync } = useSaveVariantAxis();
  const save = ({ name, values }: AxisValues) =>
    axis
      ? mutateAsync({ id: axis.id, body: { values: parseLines(values) } })
      : mutateAsync({ body: { name, values: parseLines(values) } });
  const { formError, onSubmit, close } = useSaveDialog(form, {
    fields: axisErrorFields,
    save,
    success: (saved) => ({ title: 'Eje guardado', description: saved.name }),
    onClose,
  });
  const { errors, isSubmitting } = form.formState;
  const { title, description, nameHint } = axis ? copy.edit : copy.add;

  return (
    <FormDialog
      open={open}
      onClose={close}
      title={title}
      description={description}
      size="md"
      onSubmit={onSubmit}
      isSubmitting={isSubmitting}
      formError={formError}
    >
      <Field label="Nombre" hint={nameHint} error={errors.name?.message}>
        <Input autoComplete="off" readOnly={Boolean(axis)} {...form.register('name')} />
      </Field>
      <Field label="Valores" hint="Uno por línea, en el orden en que se ofrecen." error={errors.values?.message}>
        <Textarea rows={6} {...form.register('values')} />
      </Field>
    </FormDialog>
  );
}
