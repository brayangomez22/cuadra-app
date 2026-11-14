import { useState } from 'react';
import { useFormContext, useWatch } from 'react-hook-form';
import { useFieldError } from '@/shared/forms/useFieldError';
import { Field, Select } from '@/shared/ui';
import type { Municipality } from '../../../../api/types';
import type { CustomerValues } from '../../../../model/customerForm/customerSchema';
import {
  departmentOf,
  departmentOptions,
  municipalityOptions,
} from '../../../../model/municipalities/municipalityOptions';

/**
 * The customer's municipality, from the DANE's catalog, which the electronic
 * invoice asks with the address: first the department, then its
 * municipalities. Choosing one also writes its name as the city.
 */
export function MunicipalityFields({ municipalities }: { municipalities: Municipality[] }) {
  const { control, setValue } = useFormContext<CustomerValues>();
  const errorOf = useFieldError(control, 'municipality_code');
  const code = useWatch({ control, name: 'municipality_code' });
  const [department, setDepartment] = useState(() => departmentOf(code));
  const choose = (value: string) => {
    const chosen = municipalities.find((m) => m.code === value);
    setValue('municipality_code', value, { shouldDirty: true });
    if (chosen) setValue('city', chosen.name, { shouldDirty: true });
  };
  return (
    <>
      <Field label="Departamento" optional>
        <Select
          options={departmentOptions(municipalities)}
          value={department || undefined}
          placeholder="Elija el departamento"
          onValueChange={(value) => {
            setDepartment(value);
            setValue('municipality_code', '', { shouldDirty: true });
          }}
        />
      </Field>
      <Field label="Municipio" optional error={errorOf('municipality_code')}>
        <Select
          options={municipalityOptions(municipalities, department)}
          value={code || undefined}
          placeholder="Elija el municipio"
          disabled={!department}
          onValueChange={choose}
        />
      </Field>
    </>
  );
}
