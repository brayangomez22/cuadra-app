import { variantName } from '@/features/catalog';
import { formatMoney, formatQuantity } from '@/shared/format';
import { keyOf, lineUnit } from './cartLine';
import type { CartTotals } from './cartTotals';

/**
 * What a screen reader hears when a variant is added: the variant, its
 * quantity in the sale and the new total. Undefined if the line is not there.
 */
export function addedMessage(totals: CartTotals, key: string): string | undefined {
  const line = totals.lines.find((l) => keyOf(l.line) === key)?.line;
  if (!line) return undefined;
  const { item } = line;
  return `${variantName(item.name, item.label)}: ${formatQuantity(line.quantity)} ${lineUnit(line).code}. Total ${formatMoney(totals.total)}`;
}
