import { zodResolver } from '@hookform/resolvers/zod';
import { useForm } from 'react-hook-form';
import type { CustomerSegment } from '@/features/customers';
import { Field, Input } from '@/shared/ui';
import { useSaveCustomerSegment } from '../../../../api/customerSegments';
import { FormDialog } from '../../../../components/FormDialog/FormDialog';
import { useSaveDialog } from '../../../../hooks/useSaveDialog';
import { segmentErrorFields, segmentSchema, type SegmentValues } from '../../../../model/segmentSchema';

type Props = {
  open: boolean;
  /** The segment to rename, or null to add one. */
  segment: CustomerSegment | null;
  onClose: () => void;
};

export function SegmentDialog({ open, segment, onClose }: Props) {
  const form = useForm({ resolver: zodResolver(segmentSchema), values: { name: segment?.name ?? '' } });
  const { mutateAsync } = useSaveCustomerSegment();
  const { formError, onSubmit, close } = useSaveDialog(form, {
    fields: segmentErrorFields,
    save: ({ name }: SegmentValues) => mutateAsync({ id: segment?.id, body: { name } }),
    success: (saved) => ({ title: 'Segmento guardado', description: saved.name }),
    onClose,
  });
  const { errors, isSubmitting } = form.formState;

  return (
    <FormDialog
      open={open}
      onClose={close}
      title={segment ? 'Renombrar segmento' : 'Agregar segmento'}
      description="Los clientes del segmento lo conservan con su nuevo nombre."
      size="sm"
      onSubmit={onSubmit}
      isSubmitting={isSubmitting}
      formError={formError}
    >
      <Field label="Nombre" hint="Mayorista, Convenio, Cliente frecuente…" error={errors.name?.message}>
        <Input autoComplete="off" {...form.register('name')} />
      </Field>
    </FormDialog>
  );
}
