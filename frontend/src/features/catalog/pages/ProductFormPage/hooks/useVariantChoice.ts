import { useState } from 'react';
import { useFieldArray, useWatch, type Control } from 'react-hook-form';
import type { VariantAxis } from '../../../api/types';
import type { ProductFormInput } from '../../../model/productForm/productSchema';
import { MAX_AXES, chosenOf, nextRows, toggleValue, withAxesOf } from '../../../model/variantMatrix/variantChoice';

/**
 * The values picked in the variant editor and the rows of the matrix they
 * make. A variant that exists keeps its axes and values: variants are not
 * deleted and every variant of a product names the same axes.
 */
export function useVariantChoice(control: Control<ProductFormInput>, axes: VariantAxis[]) {
  const { fields, replace } = useFieldArray({ control, name: 'variants' });
  // The rows as typed: the fields only hold what each row started with.
  const rows = useWatch({ control, name: 'variants' });
  const [chosen, setChosen] = useState(() => chosenOf(fields));
  const [tooMany, setTooMany] = useState(false);

  const existing = rows.filter((row) => row.variantId);
  const lockedAxes = existing.length > 0 ? Object.keys(existing[0]?.values ?? {}) : undefined;
  const allAxes = withAxesOf(axes, rows);
  const chosenAxes = Object.keys(chosen).filter((axis) => (chosen[axis] ?? []).length > 0);

  function toggle(axis: string, value: string) {
    const next = toggleValue(chosen, axis, value);
    const nextMatrix = nextRows(allAxes, next, rows);
    setTooMany(nextMatrix === null);
    if (nextMatrix === null) return;
    setChosen(next);
    replace(nextMatrix);
  }

  /** Whether the axis takes no more values: not one of the variants', or one too many. */
  function isAxisLocked(axis: string) {
    if (lockedAxes !== undefined && !lockedAxes.includes(axis)) return true;
    return !chosenAxes.includes(axis) && chosenAxes.length >= MAX_AXES;
  }

  /** Whether an existing variant uses the value, so it cannot be unpicked. */
  const isValueUsed = (axis: string, value: string) => existing.some((row) => row.values[axis] === value);

  return { fields, rows, chosen, allAxes, lockedAxes, tooMany, toggle, isAxisLocked, isValueUsed };
}
