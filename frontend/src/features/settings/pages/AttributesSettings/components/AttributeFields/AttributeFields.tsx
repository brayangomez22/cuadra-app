import { useFormContext, useWatch } from 'react-hook-form';
import { SelectController } from '@/shared/forms/SelectController/SelectController';
import { Checkbox, Field, Input, Textarea } from '@/shared/ui';
import { typeOptions } from '../../../../model/attributeTypes';
import type { AttributeValues } from '../../../../model/attributeSchema';

/** The attribute's fields. Editing, the key and the type are read only. */
export function AttributeFields({ editing }: { editing: boolean }) {
  const {
    register,
    control,
    formState: { errors },
  } = useFormContext<AttributeValues>();
  const type = useWatch({ control, name: 'type' });

  return (
    <>
      <Field label="Nombre" hint="Como se ve en el formulario: Principio activo" error={errors.label?.message}>
        <Input autoComplete="off" {...register('label')} />
      </Field>
      <Field
        label="Clave"
        hint={
          editing ? undefined : 'Minúsculas, sin tildes ni espacios: principio_activo. No se puede cambiar después.'
        }
        error={errors.key?.message}
      >
        <Input autoComplete="off" spellCheck={false} readOnly={editing} {...register('key')} />
      </Field>
      <Field label="Tipo" error={errors.type?.message}>
        <SelectController control={control} name="type" options={typeOptions} disabled={editing} />
      </Field>
      {type === 'list' && (
        <Field label="Opciones" hint="Una por línea, en el orden en que se ofrecen." error={errors.options?.message}>
          <Textarea rows={5} {...register('options')} />
        </Field>
      )}
      <Checkbox
        label="Obligatorio"
        hint="Todo producto debe tener un valor. Los que no lo tienen lo piden en su próxima edición."
        {...register('required')}
      />
    </>
  );
}
