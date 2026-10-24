import { zodResolver } from '@hookform/resolvers/zod';
import { FormProvider, useForm } from 'react-hook-form';
import type { AttributeDefinition } from '@/features/catalog';
import { useSaveAttribute } from '../../../../api/attributes';
import { FormDialog } from '../../../../components/FormDialog/FormDialog';
import { useSaveDialog } from '../../../../hooks/useSaveDialog';
import { toAttributeCreate, toAttributeUpdate, toAttributeValues } from '../../../../model/attributeInput';
import { attributeErrorFields, attributeSchema, type AttributeValues } from '../../../../model/attributeSchema';
import { AttributeFields } from '../AttributeFields/AttributeFields';

type Props = {
  open: boolean;
  /** The attribute to edit, or null to add one. */
  attribute: AttributeDefinition | null;
  onClose: () => void;
};

export function AttributeDialog({ open, attribute, onClose }: Props) {
  const form = useForm({ resolver: zodResolver(attributeSchema), values: toAttributeValues(attribute) });
  const { mutateAsync } = useSaveAttribute();
  const save = (values: AttributeValues) =>
    attribute
      ? mutateAsync({ id: attribute.id, body: toAttributeUpdate(values) })
      : mutateAsync({ body: toAttributeCreate(values) });
  const { formError, onSubmit, close } = useSaveDialog(form, {
    fields: attributeErrorFields,
    save,
    success: (saved) => ({ title: 'Atributo guardado', description: saved.label }),
    onClose,
  });

  return (
    <FormDialog
      open={open}
      onClose={close}
      title={attribute ? 'Editar atributo' : 'Agregar atributo'}
      description={
        attribute
          ? 'La clave y el tipo no cambian, porque los productos guardan sus valores con ellos.'
          : 'Aparece como un campo más en el formulario de producto.'
      }
      size="md"
      onSubmit={onSubmit}
      isSubmitting={form.formState.isSubmitting}
      formError={formError}
    >
      <FormProvider {...form}>
        <AttributeFields editing={attribute !== null} />
      </FormProvider>
    </FormDialog>
  );
}
