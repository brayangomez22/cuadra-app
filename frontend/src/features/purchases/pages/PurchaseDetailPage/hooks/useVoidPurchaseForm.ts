import { zodResolver } from '@hookform/resolvers/zod';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { ApiError } from '@/shared/api/errors';
import { useApiFormErrors } from '@/shared/forms/useApiFormErrors';
import { useToast } from '@/shared/ui';
import { useVoidPurchase } from '../../../api/purchases';
import type { Purchase } from '../../../api/types';
import { voidErrorMessage } from '../../../model/purchaseStatus/purchaseStatus';
import { voidPurchaseSchema, type VoidPurchaseValues } from '../../../model/purchaseStatus/voidPurchaseSchema';

const fields = { invalid_void_reason: 'reason' } as const;

/** The reason of the void and sending it; stock already gone is told by its product. */
export function useVoidPurchaseForm(purchase: Purchase, onDone: () => void) {
  const voidPurchase = useVoidPurchase(purchase.id);
  const { toast } = useToast();
  const form = useForm<VoidPurchaseValues>({
    resolver: zodResolver(voidPurchaseSchema),
    defaultValues: { reason: '' },
  });
  const { formError, handleError, clearFormError } = useApiFormErrors(form.setError, fields);
  const [stockError, setStockError] = useState<string | null>(null);

  const submit = form.handleSubmit(({ reason }) => {
    clearFormError();
    setStockError(null);
    voidPurchase.mutate(reason, {
      onSuccess: () => {
        toast({ tone: 'success', title: `Compra ${purchase.invoice_number} anulada` });
        onDone();
      },
      onError: (error) => {
        if (error instanceof ApiError && error.code === 'insufficient_stock') {
          setStockError(voidErrorMessage(purchase, error.details.line));
        } else handleError(error);
      },
    });
  });

  return { form, submit, formError: stockError ?? formError, pending: voidPurchase.isPending };
}
