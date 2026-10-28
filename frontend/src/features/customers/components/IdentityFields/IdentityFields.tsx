import { Controller, useFormContext, useWatch } from 'react-hook-form';
import { SelectController } from '@/shared/forms/SelectController/SelectController';
import { useFieldError } from '@/shared/forms/useFieldError';
import { Field, Input, Select } from '@/shared/ui';
import { documentTypeOptions, personTypeOptions } from '../../model/customerForm/documentTypes';
import type { CustomerValues } from '../../model/customerForm/customerSchema';

/**
 * Who the customer is: the kind of person, the document and the name, which
 * the full form and the POS's quick creation share. A legal person only
 * offers NIT, and its name is the company's. autoFocus is the field to
 * start typing in, in a dialog.
 */
export function IdentityFields({ autoFocus }: { autoFocus?: 'document_number' | 'name' }) {
  const { control, register, setValue } = useFormContext<CustomerValues>();
  const errorOf = useFieldError(control);
  const person = useWatch({ control, name: 'person_type' });
  const documentType = useWatch({ control, name: 'document_type' });
  const legal = person === 'legal';
  return (
    <>
      <Field label="Tipo de persona" error={errorOf('person_type')}>
        <Controller
          control={control}
          name="person_type"
          render={({ field }) => (
            <Select
              options={personTypeOptions}
              value={field.value}
              onValueChange={(value) => {
                field.onChange(value);
                if (value === 'legal') setValue('document_type', 'nit');
              }}
            />
          )}
        />
      </Field>
      <Field label="Tipo de documento" error={errorOf('document_type')}>
        <SelectController control={control} name="document_type" options={documentTypeOptions(person)} />
      </Field>
      <Field
        label="Número de documento"
        hint={documentType === 'nit' ? 'Con el dígito de verificación, por ejemplo 900123456-8' : undefined}
        error={errorOf('document_number')}
      >
        <Input
          autoComplete="off"
          spellCheck={false}
          autoFocus={autoFocus === 'document_number'}
          {...register('document_number')}
        />
      </Field>
      <Field label={legal ? 'Razón social' : 'Nombre'} error={errorOf('name')}>
        <Input autoComplete="off" autoFocus={autoFocus === 'name'} {...register('name')} />
      </Field>
    </>
  );
}
