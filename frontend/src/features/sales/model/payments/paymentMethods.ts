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
};
