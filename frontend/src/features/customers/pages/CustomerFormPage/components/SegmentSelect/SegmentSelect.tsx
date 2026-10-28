import { Controller, useFormContext } from 'react-hook-form';
import { useFieldError } from '@/shared/forms/useFieldError';
import { Field, Select } from '@/shared/ui';
import type { CustomerSegment } from '../../../../api/types';
import type { CustomerValues } from '../../../../model/customerForm/customerSchema';

// A Select has no empty value: this one stands for "without a segment".
const NONE = 'ninguno';

/** The customer's segment, one of the tenant's, or none. */
export function SegmentSelect({ segments }: { segments: CustomerSegment[] }) {
  const { control } = useFormContext<CustomerValues>();
  const errorOf = useFieldError(control, 'segment_id');
  const options = [{ value: NONE, label: 'Sin segmento' }, ...segments.map((s) => ({ value: s.id, label: s.name }))];
  return (
    <Field label="Segmento" optional hint="Se crean en Configuración." error={errorOf('segment_id')}>
      <Controller
        control={control}
        name="segment_id"
        render={({ field }) => (
          <Select
            options={options}
            value={field.value || NONE}
            onValueChange={(value) => field.onChange(value === NONE ? '' : value)}
          />
        )}
      />
    </Field>
  );
}
