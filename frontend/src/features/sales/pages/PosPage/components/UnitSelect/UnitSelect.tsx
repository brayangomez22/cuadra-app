import type { Dispatch, ReactNode } from 'react';
import { formatQuantity } from '@/shared/format';
import { Field, Select } from '@/shared/ui';
import type { CartLine } from '../../../../model/cart/cartLine';
import type { CartAction } from '../../../../model/cart/cartReducer';
import styles from './UnitSelect.module.scss';

type Props = { line: CartLine; label: ReactNode; dispatch: Dispatch<CartAction>; onDone: () => void };

// The Select's value for the product's unit: a unit id is never empty.
const BASE_UNIT = 'base';

/** The unit a line is sold in: the product's or one of the variant's alternate units. */
export function UnitSelect({ line, label, dispatch, onDone }: Props) {
  return (
    <Field className={styles.root} label={label}>
      <Select
        options={unitOptions(line)}
        value={line.unitId ?? BASE_UNIT}
        onValueChange={(value) => {
          const next = value === BASE_UNIT ? undefined : value;
          dispatch({ type: 'setUnit', variantId: line.item.variantId, unitId: line.unitId, next });
          onDone();
        }}
      />
    </Field>
  );
}

// unitOptions offers the product's unit and each alternate one with what it
// holds: "caja (12 und)".
function unitOptions({ item }: CartLine) {
  return [
    { value: BASE_UNIT, label: item.unit },
    ...item.units.map((u) => ({ value: u.id, label: `${u.code} (${formatQuantity(u.factor)} ${item.unit})` })),
  ];
}
