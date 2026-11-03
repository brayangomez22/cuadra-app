import { variantName, type Sellable } from '@/features/catalog';
import { formatQuantity, isNegative } from '@/shared/format';

const nameOf = ({ product, variant }: Sellable) => variantName(product.name, variant.label);

type Adjusted = { item: Sellable; delta: string; balanceAfter: string; locationName: string };

/** What the toast says after an adjustment. */
export function adjustedMessage({ item, delta, balanceAfter, locationName }: Adjusted) {
  const unit = item.product.unit.code;
  const verb = isNegative(delta) ? 'Salen' : 'Entran';
  const amount = formatQuantity(delta.replace('-', ''));
  return `${verb} ${amount} ${unit} de ${nameOf(item)}. ${locationName} queda con ${formatQuantity(balanceAfter)} ${unit}.`;
}

type Transferred = {
  item: Sellable;
  quantity: string;
  from: { name: string; balance: string };
  to: { name: string; balance: string };
};

/** What the toast says after a transfer. */
export function transferredMessage({ item, quantity, from, to }: Transferred) {
  const unit = item.product.unit.code;
  const left = `${from.name} queda con ${formatQuantity(from.balance)} ${unit}`;
  return `${formatQuantity(quantity)} ${unit} de ${nameOf(item)}. ${left} y ${to.name} con ${formatQuantity(to.balance)} ${unit}.`;
}

/** insufficient_stock says neither what nor where; the form knows both. */
export function insufficientStockMessage(item: Sellable, locationName: string) {
  return `No hay stock suficiente de ${nameOf(item)} en ${locationName}.`;
}
