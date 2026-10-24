import { useFormContext } from 'react-hook-form';
import { Checkbox, Field, Input } from '@/shared/ui';
import type { AttributeDefinition } from '../../../../api/types';
import { SelectController } from '@/shared/forms/SelectController/SelectController';
import { useFieldError } from '@/shared/forms/useFieldError';
import { NO_VALUE, type ProductFormInput } from '../../../../model/productForm/productSchema';
import { FormGridWide } from '../FormSection/FormSection';

const numeric = { inputMode: 'decimal', align: 'end' } as const;

/** The options of a list attribute; an optional one can be left without value. */
function listOptions(definition: AttributeDefinition) {
  const options = definition.options.map((option) => ({ value: option, label: option }));
  return definition.required ? options : [{ value: NO_VALUE, label: 'Sin valor' }, ...options];
}

/** The field of one attribute the tenant defines, by its type. The optional ones say so. */
export function AttributeField({ definition }: { definition: AttributeDefinition }) {
  const { register, control } = useFormContext<ProductFormInput>();
  const name = `attributes.${definition.key}` as const;
  const error = useFieldError(control, name)(name);
  if (definition.type === 'boolean') {
    return (
      <FormGridWide>
        <Checkbox label={definition.label} {...register(name)} />
      </FormGridWide>
    );
  }
  return (
    <Field label={definition.label} optional={!definition.required} error={error}>
      {definition.type === 'list' ? (
        <SelectController
          control={control}
          name={name}
          options={listOptions(definition)}
          placeholder="Elija una opción"
        />
      ) : (
        <Input {...(definition.type === 'number' ? numeric : {})} autoComplete="off" {...register(name)} />
      )}
    </Field>
  );
}
