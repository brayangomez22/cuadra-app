import type { components } from '@/shared/api/schema';

export type PaymentMethod = components['schemas']['PaymentMethod'];
export type PaymentInput = components['schemas']['PaymentInput'];

/** How the cashier and the receipt name each payment method. */
export const methodLabels: Record<PaymentMethod, string> = {
  cash: 'Efectivo',
  card: 'Tarjeta',
  nequi: 'Nequi',
  daviplata: 'Daviplata',
  transfer: 'Transferencia',
  credit: 'Crédito',
};

const paidNow: PaymentMethod[] = ['cash', 'card', 'nequi', 'daviplata', 'transfer'];

/** The methods the POS offers to pay with. Credit only for a customer with a credit limit. */
export function posMethods(withCredit: boolean): PaymentMethod[] {
  return withCredit ? [...paidNow, 'credit'] : paidNow;
}
