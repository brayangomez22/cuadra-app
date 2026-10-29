import { formatQuantity } from '@/shared/format';
import type { SelectOption } from '@/shared/ui';
import type { Sellable } from './variants';

/** The option of the product's unit: a unit id is never this, so an empty unit_id can show as it. */
export const BASE_UNIT = 'base';

/** The units a quantity of the variant can be in: the product's and each alternate one, "caja (12 und)". */
export function unitOptions(item: Sellable): SelectOption[] {
  const base = item.product.unit.code;
  return [
    { value: BASE_UNIT, label: base },
    ...item.variant.units.map((u) => ({
      value: u.unit.id,
      label: `${u.unit.code} (${formatQuantity(u.factor)} ${base})`,
    })),
  ];
}
