import { useFormContext, useWatch } from 'react-hook-form';
import { BASE_UNIT, unitOptions } from '@/features/catalog';
import { useModule } from '@/features/auth';
import { useFieldError } from '@/shared/forms/useFieldError';
import { Field, Select } from '@/shared/ui';
import type { PurchaseInputValues } from '../../../../model/purchaseForm/purchaseSchema';

/** The unit of purchase of a line, only with unit_conversions and when its variant has alternate units. */
export function LineUnitField({ index }: { index: number }) {
  const { control, setValue } = useFormContext<PurchaseInputValues>();
  const alternateUnits = useModule('unit_conversions');
  const [item, unitId] = useWatch({ control, name: [`lines.${index}.item`, `lines.${index}.unit_id`] });
  const errorOf = useFieldError(control, `lines.${index}.unit_id`);
  if (!alternateUnits || !item || item.variant.units.length === 0) return null;
  return (
    <Field label="Unidad" error={errorOf(`lines.${index}.unit_id`)}>
      <Select
        options={unitOptions(item)}
        value={unitId || BASE_UNIT}
        onValueChange={(value) => setValue(`lines.${index}.unit_id`, value === BASE_UNIT ? '' : value)}
      />
    </Field>
  );
}
