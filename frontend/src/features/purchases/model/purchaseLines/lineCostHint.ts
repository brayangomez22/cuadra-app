import type { Sellable } from '@/features/catalog';

/** The hint of a line's cost: it is without VAT and per unit of purchase, by its name: "Sin IVA, por galón." */
export function lineCostHint(item: Sellable | null, unitId: string): string {
  if (!item) return 'Sin IVA, por unidad de compra.';
  const alternate = item.variant.units.find((u) => u.unit.id === unitId);
  const unit = alternate?.unit ?? item.product.unit;
  return `Sin IVA, por ${unit.name.toLowerCase()}.`;
}
