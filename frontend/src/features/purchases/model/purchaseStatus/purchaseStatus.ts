import { variantName } from '@/features/catalog';
import { toUnits } from '@/shared/money/decimal';
import type { Purchase, PurchaseSummary } from '../../api/types';

type Status = PurchaseSummary['status'];

/** The badge's color of a status; its text says the status on its own. */
export function statusTone(status: Status): 'warning' | 'success' | 'neutral' {
  if (status === 'open') return 'warning';
  return status === 'paid' ? 'success' : 'neutral';
}

/**
 * Whether the purchase can be voided: once a supplier was paid part of a
 * purchase on credit, voiding it would leave the payment without a purchase
 * (the API refuses it too).
 */
export function canVoid(purchase: Purchase): boolean {
  if (purchase.status === 'voided') return false;
  return purchase.payment === 'cash' || toUnits(purchase.balance_due) === toUnits(purchase.total);
}

/** Whether something is left to pay to the supplier. */
export function canPay(purchase: Purchase): boolean {
  return purchase.status === 'open';
}

/** insufficient_stock of a void names its line (details.line, from 0): the form says which product. */
export function voidErrorMessage(purchase: Purchase, line: string | undefined): string {
  const l = line === undefined ? undefined : purchase.lines[Number(line)];
  const what = l ? `de ${variantName(l.product_name, l.variant_label ?? '')}` : 'con la compra';
  return `No se puede anular: ya no está en la sede todo lo que entró ${what}. Se vendió o se movió.`;
}
