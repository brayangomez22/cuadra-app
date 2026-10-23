// What the variant editor offers: the values picked by axis and the rows of
// the matrix they make (docs/decisiones.md, T24i).
import type { VariantAxis } from '../api/types';
import type { VariantRow } from './productSchema';
import { combinations, comboKey } from './variantMatrix';

/** The values picked, by axis name. */
export type Chosen = Record<string, string[]>;

// The API's limits: up to 5 axes per product. The matrix stops at 100
// combinations, a sane number of variants to create at once.
export const MAX_AXES = 5;
export const MAX_COMBINATIONS = 100;

/** The values the rows use, by axis. */
export function chosenOf(rows: VariantRow[]): Chosen {
  const chosen: Chosen = {};
  for (const row of rows) {
    for (const [axis, value] of Object.entries(row.values)) {
      const values = (chosen[axis] ??= []);
      if (!values.includes(value)) values.push(value);
    }
  }
  return chosen;
}

/**
 * The tenant's axes plus the axes and values its variants use that the
 * tenant no longer defines, so they show and keep their variants.
 */
export function withAxesOf(axes: VariantAxis[], rows: VariantRow[]): VariantAxis[] {
  const chosen = chosenOf(rows);
  const known = axes.map((axis) => ({
    ...axis,
    values: [...axis.values, ...(chosen[axis.name] ?? []).filter((value) => !axis.values.includes(value))],
  }));
  const unknown = Object.keys(chosen)
    .filter((name) => !axes.some((axis) => axis.name === name))
    .map((name) => ({ id: name, name, values: chosen[name] ?? [] }));
  return [...known, ...unknown];
}

/** The choice with the value of the axis picked, or unpicked if it was. */
export function toggleValue(chosen: Chosen, axis: string, value: string): Chosen {
  const current = chosen[axis] ?? [];
  return { ...chosen, [axis]: current.includes(value) ? current.filter((v) => v !== value) : [...current, value] };
}

/**
 * One row per combination of the choice: a row the matrix has keeps what was
 * typed in it. Null when they would be more than MAX_COMBINATIONS.
 */
export function nextRows(axes: VariantAxis[], chosen: Chosen, rows: VariantRow[]): VariantRow[] | null {
  const combos = combinations(axes, chosen);
  if (combos.length > MAX_COMBINATIONS) return null;
  const byKey = new Map(rows.map((row) => [comboKey(row.values), row]));
  return combos.map(
    (values): VariantRow =>
      byKey.get(comboKey(values)) ?? { values, sku: '', barcode: '', cost: '', price: '', active: true },
  );
}
