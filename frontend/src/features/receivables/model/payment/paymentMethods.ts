import type { CustomerPaymentMethod } from '../../api/types';

/** The methods a customer pays a debt with: never with credit. */
export const paymentMethodLabels: Record<CustomerPaymentMethod, string> = {
  cash: 'Efectivo',
  card: 'Tarjeta',
  nequi: 'Nequi',
  daviplata: 'Daviplata',
  transfer: 'Transferencia',
};

export const paymentMethodOptions = Object.entries(paymentMethodLabels).map(([value, label]) => ({ value, label }));
