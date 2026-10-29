import type { Sellable } from '@/features/catalog';
import { formatQuantity, parseDecimalInput } from '@/shared/format';
import { fromUnitsTrimmed, toUnits } from '@/shared/money/decimal';

type Line = { item: Sellable | null; unitId: string; quantity: string };

// toUnits keeps 4 decimals: a product of two of them has 8.
const SCALE = 10n ** 4n;

/**
 * What a line in an alternate unit adds to the stock, in the product's unit:
 * "Entran 24 und." Without a quantity yet, how many each unit brings. In the
 * product's unit there is nothing to convert: null.
 */
export function lineReceived({ item, unitId, quantity }: Line): string | null {
  const alternate = item?.variant.units.find((u) => u.unit.id === unitId);
  if (!item || !alternate) return null;
  const base = item.product.unit.code;
  const typed = parseDecimalInput(quantity);
  if (!typed.ok) return `Cada ${alternate.unit.code} trae ${formatQuantity(alternate.factor)} ${base}.`;
  const received = (toUnits(typed.value) * toUnits(alternate.factor)) / SCALE;
  return `Entran ${formatQuantity(fromUnitsTrimmed(received))} ${base}.`;
}
