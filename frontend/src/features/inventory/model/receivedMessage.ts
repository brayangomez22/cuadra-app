import { variantName, type Sellable } from '@/features/catalog';
import { formatQuantity } from '@/shared/format';
import type { StockMovement } from '../api/types';

type Received = {
  item: Sellable;
  /** The alternate unit the goods arrived in; empty is the product's unit. */
  unitId: string;
  /** As typed, in that unit. */
  quantity: string;
  movement: StockMovement;
  locationName: string;
};

/** What the toast says after an entry. The movement is in the base unit: an entry in boxes says both. */
export function receivedMessage({ item: { product, variant }, unitId, quantity, movement, locationName }: Received) {
  const base = product.unit.code;
  const received = variant.units.find((u) => u.unit.id === unitId);
  const amount = received
    ? `${formatQuantity(quantity)} ${received.unit.code} (${formatQuantity(movement.quantity)} ${base})`
    : `${formatQuantity(movement.quantity)} ${base}`;
  const name = variantName(product.name, variant.label);
  return `${amount} de ${name}. ${locationName} queda con ${formatQuantity(movement.balance_after)} ${base}.`;
}
