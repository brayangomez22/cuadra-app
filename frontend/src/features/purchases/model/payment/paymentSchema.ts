import { z } from 'zod';
import type { FieldsByErrorCode } from '@/shared/forms/useApiFormErrors';
import { moneyField } from '@/shared/forms/moneyField';
import type { SupplierPaymentMethod } from '../../api/types';

// The payment to a supplier; the API checks it against what is owed and the invoice's date.
export const paymentSchema = z.object({
  method: z.custom<SupplierPaymentMethod | ''>().refine(Boolean, 'Elija el medio de pago'),
  amount: moneyField('positive', 'Escriba el valor del pago'),
  paid_on: z.string().min(1, 'Elija la fecha del pago'),
});

export type PaymentFormInput = z.input<typeof paymentSchema>;
export type PaymentValues = z.output<typeof paymentSchema>;

// Error codes of the API (api/openapi.yaml) that belong to one field.
export const paymentErrorFields: FieldsByErrorCode<PaymentFormInput> = {
  payment_exceeds_balance: 'amount',
  invalid_payment_amount: 'amount',
  invalid_payment_method: 'method',
  payment_date_in_future: 'paid_on',
  payment_before_invoice: 'paid_on',
};
