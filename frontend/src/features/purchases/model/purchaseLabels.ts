import type { PurchasePayment, PurchaseSummary } from '../api/types';

export const paymentLabels: Record<PurchasePayment, string> = { cash: 'Contado', credit: 'Crédito' };

export const statusLabels: Record<PurchaseSummary['status'], string> = { open: 'Por pagar', paid: 'Pagada' };
