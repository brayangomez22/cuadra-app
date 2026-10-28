import { useFormContext, useWatch } from 'react-hook-form';
import type { Sellable } from '@/features/catalog';
import { useFieldError } from '@/shared/forms/useFieldError';
import { formatQuantity } from '@/shared/format';
import { Field, Select } from '@/shared/ui';
import type { ReceiptInput } from '../../../../model/receipt/receiptSchema';

// The Select's value for the product's unit: a unit id is never empty.
const BASE_UNIT = 'base';

/** The unit the goods arrive in, only when the variant has alternate units. */
export function UnitField({ onChange }: { onChange: (item: Sellable | null, unitId: string) => void }) {
  const { control } = useFormContext<ReceiptInput>();
  const item = useWatch({ control, name: 'item' });
  const unitId = useWatch({ control, name: 'unit_id' });
  const errorOf = useFieldError(control, 'unit_id');
  if (!item || item.variant.units.length === 0) return null;

  const base = item.product.unit.code;
  const options = [
    { value: BASE_UNIT, label: base },
    ...item.variant.units.map((u) => ({
      value: u.unit.id,
      label: `${u.unit.code} (${formatQuantity(u.factor)} ${base})`,
    })),
  ];
  return (
    <Field label="Unidad" hint="En la que llega la mercancía." error={errorOf('unit_id')}>
      <Select
        options={options}
        value={unitId || BASE_UNIT}
        onValueChange={(value) => onChange(item, value === BASE_UNIT ? '' : value)}
      />
    </Field>
  );
}
