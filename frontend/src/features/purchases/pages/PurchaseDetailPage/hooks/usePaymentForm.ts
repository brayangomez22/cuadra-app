import { zodResolver } from '@hookform/resolvers/zod';
import { useForm } from 'react-hook-form';
import { useApiFormErrors } from '@/shared/forms/useApiFormErrors';
import { formatMoney, toMoneyInput, todayInBogota } from '@/shared/format';
import { useToast } from '@/shared/ui';
import { useRecordSupplierPayment } from '../../../api/supplierPayments';
import type { Purchase } from '../../../api/types';
import {
  paymentErrorFields,
  paymentSchema,
  type PaymentFormInput,
  type PaymentValues,
} from '../../../model/payment/paymentSchema';
import { usePaymentCashOut } from './usePaymentCashOut';

/** A payment to the supplier: by default what is left to pay, today; in cash, out of the user's open drawer. */
export function usePaymentForm(purchase: Purchase, supplierName: string, onDone: () => void) {
  const record = useRecordSupplierPayment(purchase.id);
  const { toast } = useToast();
  const form = useForm<PaymentFormInput, unknown, PaymentValues>({
    resolver: zodResolver(paymentSchema),
    defaultValues: { method: '', amount: toMoneyInput(purchase.balance_due), paid_on: todayInBogota() },
  });
  const cashOut = usePaymentCashOut({ purchase, supplierName, control: form.control });
  const { formError, handleError, clearFormError } = useApiFormErrors(form.setError, paymentErrorFields);

  // After the payment: the cash out, if chosen, then the toast. A failed cash
  // out has its own toast, which says the payment was registered.
  const finish = async (amount: string, balanceAfter: string) => {
    if (await cashOut.afterPayment(amount)) {
      toast({
        tone: 'success',
        title: 'Pago registrado',
        description: `Queda por pagar ${formatMoney(balanceAfter)}.`,
      });
    }
    onDone();
  };

  const submit = form.handleSubmit(({ method, amount, paid_on }) => {
    if (!method) return;
    clearFormError();
    record.mutate(
      { method, amount, paid_on },
      {
        onSuccess: (payment) => void finish(amount, payment.balance_after),
        onError: handleError,
      },
    );
  });

  return { form, submit, formError, cashOut, pending: record.isPending };
}
