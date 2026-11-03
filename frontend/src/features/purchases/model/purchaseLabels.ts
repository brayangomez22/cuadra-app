import type { PurchasePayment, PurchaseSummary, SupplierPaymentMethod } from '../api/types';

export const paymentLabels: Record<PurchasePayment, string> = { cash: 'Contado', credit: 'Crédito' };

export const statusLabels: Record<PurchaseSummary['status'], string> = {
  open: 'Por pagar',
  paid: 'Pagada',
  voided: 'Anulada',
};

export const paymentMethodLabels: Record<SupplierPaymentMethod, string> = {
  cash: 'Efectivo',
  transfer: 'Transferencia',
  card: 'Tarjeta',
  nequi: 'Nequi',
  daviplata: 'Daviplata',
};
