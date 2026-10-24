// The variant matrix of the product form: the combinations of the axis
// values the person picks, one variant each (docs/decisiones.md, T24i).
import type { Product, VariantAxis } from '../../api/types';

/** Axis name → value, as the API's axis_values. */
export type AxisValues = Record<string, string>;

/**
 * "Rojo · M": the values sorted by axis name, as the API names a variant
 * (ProductVariant.label) and a sale freezes it (variant_label).
 */
export function axisLabel(values: AxisValues): string {
  return Object.keys(values)
    .sort()
    .map((axis) => values[axis])
    .join(' · ');
}

/**
 * Every combination of the chosen values, in the tenant's order of axes and
 * values. An axis with no value chosen does not take part.
 */
export function combinations(axes: VariantAxis[], chosen: Record<string, string[]>): AxisValues[] {
  let result: AxisValues[] = [];
  for (const axis of axes) {
    const values = orderedValues(axis, chosen[axis.name] ?? []);
    if (values.length === 0) continue;
    result =
      result.length === 0
        ? values.map((value) => ({ [axis.name]: value }))
        : result.flatMap((combo) => values.map((value) => ({ ...combo, [axis.name]: value })));
  }
  return result;
}

// The chosen values in the axis' order; a value the axis no longer lists
// (an old variant's) goes last.
function orderedValues(axis: VariantAxis, chosen: string[]) {
  const known = axis.values.filter((value) => chosen.includes(value));
  return [...known, ...chosen.filter((value) => !axis.values.includes(value))];
}

/** A combination's key, the same whatever the order of its axes. */
export function comboKey(values: AxisValues): string {
  return JSON.stringify(Object.entries(values).sort(([a], [b]) => a.localeCompare(b)));
}

/**
 * The SKU proposed for a variant: the base and its values in the axes' order
 * ("CAM-M-AZUL"), uppercase without spaces or accents. Empty without a base.
 */
export function generatedSku(base: string, values: AxisValues): string {
  const part = (text: string) =>
    text
      .normalize('NFD')
      .replace(/\p{Diacritic}/gu, '')
      .replace(/\s+/g, '')
      .toUpperCase();
  const trimmed = part(base);
  return trimmed ? [trimmed, ...Object.values(values).map(part)].join('-') : '';
}

/** The base of the SKU of a product with variants: what their SKUs share ("CAM" of CAM-S-ROJO and CAM-M-AZUL). */
export function skuPrefix(product: Product): string {
  const skus = product.variants.map((v) => v.sku);
  let prefix = skus[0] ?? '';
  for (const sku of skus) {
    while (!sku.startsWith(prefix)) prefix = prefix.slice(0, -1);
  }
  return prefix.replace(/[^\p{L}\p{N}]+$/u, '');
}
