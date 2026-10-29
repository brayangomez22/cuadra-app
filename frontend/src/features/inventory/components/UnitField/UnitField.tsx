import { useFormContext, useWatch } from 'react-hook-form';
import { BASE_UNIT, unitOptions, type Sellable } from '@/features/catalog';
import { useFieldError } from '@/shared/forms/useFieldError';
import { Field, Select } from '@/shared/ui';

/** The fields UnitField reads: the chosen variant and its unit (empty: the product's). */
type UnitFieldValues = { item: Sellable | null; unit_id: string };

type Props = {
  /** What the unit is for: "En la que llega la mercancía." */
  hint: string;
  onChange: (item: Sellable | null, unitId: string) => void;
};

/** The unit a quantity is in, only when the variant has alternate units. */
export function UnitField({ hint, onChange }: Props) {
  const { control } = useFormContext<UnitFieldValues>();
  const item = useWatch({ control, name: 'item' });
  const unitId = useWatch({ control, name: 'unit_id' });
  const errorOf = useFieldError(control, 'unit_id');
  if (!item || item.variant.units.length === 0) return null;

  return (
    <Field label="Unidad" hint={hint} error={errorOf('unit_id')}>
      <Select
        options={unitOptions(item)}
        value={unitId || BASE_UNIT}
        onValueChange={(value) => onChange(item, value === BASE_UNIT ? '' : value)}
      />
    </Field>
  );
}
