import { Controller, type Control, type FieldValues, type Path } from 'react-hook-form';
import { Select, type SelectOption } from '@/shared/ui';

type Props<T extends FieldValues> = {
  control: Control<T>;
  name: Path<T>;
  options: SelectOption[];
  placeholder?: string;
  disabled?: boolean;
};

/** A Select bound to a field of react-hook-form. An empty value shows the placeholder. */
export function SelectController<T extends FieldValues>({ control, name, ...select }: Props<T>) {
  return (
    <Controller
      control={control}
      name={name}
      render={({ field }) => (
        <Select
          {...select}
          value={typeof field.value === 'string' && field.value !== '' ? field.value : undefined}
          onValueChange={field.onChange}
        />
      )}
    />
  );
}
