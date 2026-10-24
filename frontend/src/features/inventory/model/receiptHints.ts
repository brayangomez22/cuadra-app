import type { Sellable } from '@/features/catalog';
import { formatQuantity } from '@/shared/format';

/** The hints of the quantity and the cost: both are in the unit the goods arrive in. */
export function receiptHints(item: Sellable | null, unitId: string): { quantity: string; cost: string } {
  if (!item) return { quantity: 'En la unidad del producto.', cost: 'Sin IVA, por unidad.' };
  const base = item.product.unit.code;
  const alternate = item.variant.units.find((u) => u.unit.id === unitId);
  if (!alternate) return { quantity: `En ${base}.`, cost: `Sin IVA, por ${base}.` };
  return {
    quantity: `En ${alternate.unit.code} de ${formatQuantity(alternate.factor)} ${base}.`,
    cost: `Sin IVA, por ${alternate.unit.code}.`,
  };
}
