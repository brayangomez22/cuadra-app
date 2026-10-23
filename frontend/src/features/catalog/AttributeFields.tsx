import { Controller, type Control, type FieldErrors, type UseFormRegister } from 'react-hook-form';
import { Checkbox, Field, Input, Select } from '@/shared/ui';
import type { AttributeDefinition } from './api';
import { NO_VALUE, type ProductFormInput } from './schemas';

type Props = {
  definitions: AttributeDefinition[];
  register: UseFormRegister<ProductFormInput>;
  control: Control<ProductFormInput>;
  errors: FieldErrors<ProductFormInput>['attributes'];
  className?: string;
  wideClassName?: string;
};

/**
 * The fields of the attributes the tenant defines for its products, by type:
 * text, a number, one option of a list or yes/no. The optional ones say so.
 */
export function AttributeFields({ definitions, register, control, errors, className, wideClassName }: Props) {
  return (
    <div className={className}>
      {definitions.map((definition) => {
        const name = `attributes.${definition.key}` as const;
        const error = errors?.[definition.key]?.message;
        const optional = !definition.required;
        switch (definition.type) {
          case 'boolean':
            return (
              <Checkbox key={definition.key} label={definition.label} className={wideClassName} {...register(name)} />
            );
          case 'list':
            return (
              <Field key={definition.key} label={definition.label} optional={optional} error={error}>
                <Controller
                  control={control}
                  name={name}
                  render={({ field }) => (
                    <Select
                      options={[
                        ...(optional ? [{ value: NO_VALUE, label: 'Sin valor' }] : []),
                        ...definition.options.map((option) => ({ value: option, label: option })),
                      ]}
                      value={typeof field.value === 'string' && field.value !== '' ? field.value : undefined}
                      onValueChange={field.onChange}
                      placeholder="Elija una opción"
                    />
                  )}
                />
              </Field>
            );
          case 'number':
            return (
              <Field key={definition.key} label={definition.label} optional={optional} error={error}>
                <Input inputMode="decimal" align="end" autoComplete="off" {...register(name)} />
              </Field>
            );
          case 'text':
            return (
              <Field key={definition.key} label={definition.label} optional={optional} error={error}>
                <Input autoComplete="off" {...register(name)} />
              </Field>
            );
        }
      })}
    </div>
  );
}
