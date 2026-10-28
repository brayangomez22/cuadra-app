import { suggestedUnitPrice, type Sellable } from '@/features/catalog';

/**
 * The last cost of the variant, times the factor of the unit the goods arrive
 * in, as a start for the entry. Only those who see costs (catalog:write) get one.
 */
export function suggestedCost({ variant }: Sellable, unitId: string): string | undefined {
  if (!variant.cost) return undefined;
  const factor = variant.units.find((u) => u.unit.id === unitId)?.factor;
  return factor ? suggestedUnitPrice(variant.cost, factor) : variant.cost;
}
