import { zodResolver } from '@hookform/resolvers/zod';
import { useForm } from 'react-hook-form';
import { useApiFormErrors } from '@/shared/forms/useApiFormErrors';
import { formatMoney } from '@/shared/format';
import { useToast } from '@/shared/ui';
import { useRecordPayment } from '../../../api/payments';
import { paymentSchema, type PaymentValues } from '../../../model/payment/paymentSchema';

type Options = { customerId: string; balance: string; onDone: () => void };

const defaults: PaymentValues = { method: 'cash', amount: '' };

/** A payment of what the customer owes. `onDone` closes the dialog once it is recorded. */
export function usePaymentForm({ customerId, balance, onDone }: Options) {
  const { toast } = useToast();
  const record = useRecordPayment(customerId);
  const form = useForm({ resolver: zodResolver(paymentSchema(balance)), defaultValues: defaults });
  const { formError, handleError, clearFormError } = useApiFormErrors(form.setError, {
    invalid_payment_amount: 'amount',
    payment_exceeds_balance: 'amount',
  });

  const submit = form.handleSubmit((values) => {
    clearFormError();
    record.mutate(values, {
      onSuccess: (payment) => {
        toast({
          tone: 'success',
          title: 'Abono registrado',
          description: `Queda debiendo ${formatMoney(payment.balance)}`,
        });
        onDone();
      },
      onError: handleError,
    });
  });

  // Each opening starts empty, in cash.
  function reset() {
    form.reset(defaults);
    clearFormError();
  }

  return { form, formError, submit, reset, isSaving: record.isPending };
}
