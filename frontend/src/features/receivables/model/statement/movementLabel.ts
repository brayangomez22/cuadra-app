import type { CreditMovement } from '../../api/types';
import { paymentMethodLabels } from '../payment/paymentMethods';

/** How the statement names a movement: a credit sale, a payment and its method, or a credit sale voided. */
export function movementLabel({ kind, method }: Pick<CreditMovement, 'kind' | 'method'>): string {
  if (kind === 'charge') return 'Venta a crédito';
  if (kind === 'cancellation') return 'Venta a crédito anulada';
  return method ? `Abono · ${paymentMethodLabels[method]}` : 'Abono';
}
