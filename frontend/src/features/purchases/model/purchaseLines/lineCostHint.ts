import type { Sellable } from '@/features/catalog';

/** The hint of a line's cost: it is without VAT and per unit of purchase, "Sin IVA, por caja." */
export function lineCostHint(item: Sellable | null, unitId: string): string {
  if (!item) return 'Sin IVA, por unidad de compra.';
  const alternate = item.variant.units.find((u) => u.unit.id === unitId);
  return `Sin IVA, por ${alternate?.unit.code ?? item.product.unit.code}.`;
}
