import { zodResolver } from '@hookform/resolvers/zod';
import { useForm } from 'react-hook-form';
import { useApiFormErrors } from '@/shared/forms/useApiFormErrors';
import { formatMoney, toDecimalInput, todayInBogota } from '@/shared/format';
import { useToast } from '@/shared/ui';
import { useRecordSupplierPayment } from '../../../api/supplierPayments';
import type { Purchase } from '../../../api/types';
import {
  paymentErrorFields,
  paymentSchema,
  type PaymentFormInput,
  type PaymentValues,
} from '../../../model/payment/paymentSchema';

/** A payment to the supplier: by default what is left to pay, today. */
export function usePaymentForm(purchase: Purchase, onDone: () => void) {
  const record = useRecordSupplierPayment(purchase.id);
  const { toast } = useToast();
  const form = useForm<PaymentFormInput, unknown, PaymentValues>({
    resolver: zodResolver(paymentSchema),
    defaultValues: { method: '', amount: toDecimalInput(purchase.balance_due), paid_on: todayInBogota() },
  });
  const { formError, handleError, clearFormError } = useApiFormErrors(form.setError, paymentErrorFields);

  const submit = form.handleSubmit(({ method, amount, paid_on }) => {
    if (!method) return;
    clearFormError();
    record.mutate(
      { method, amount, paid_on },
      {
        onSuccess: (payment) => {
          toast({
            tone: 'success',
            title: 'Pago registrado',
            description: `Queda por pagar ${formatMoney(payment.balance_after)}.`,
          });
          onDone();
        },
        onError: handleError,
      },
    );
  });

  return { form, submit, formError, pending: record.isPending };
}
