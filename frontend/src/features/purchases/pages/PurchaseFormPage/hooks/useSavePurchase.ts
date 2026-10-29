import type { UseFormReturn } from 'react-hook-form';
import { useNavigate } from 'react-router';
import { paths } from '@/app/routes/paths';
import { describeError } from '@/shared/api/errors';
import { useApiFormErrors } from '@/shared/forms/useApiFormErrors';
import { useToast } from '@/shared/ui';
import { useRegisterPurchase } from '../../../api/purchases';
import { lineErrorOf } from '../../../model/purchaseForm/lineErrorOf';
import {
  lineErrorField,
  purchaseErrorFields,
  type PurchaseInputValues,
  type PurchaseValues,
} from '../../../model/purchaseForm/purchaseSchema';
import { toPurchaseInput } from '../../../model/purchaseForm/toPurchaseInput';

type PurchaseForm = UseFormReturn<PurchaseInputValues, unknown, PurchaseValues>;

/**
 * Registers the purchase and opens it. An error of a line goes to its row
 * (details.line); any other to its field of the invoice or above the form.
 */
export function useSavePurchase(form: PurchaseForm) {
  const navigate = useNavigate();
  const { toast } = useToast();
  const { mutateAsync } = useRegisterPurchase();
  const { formError, handleError, clearFormError } = useApiFormErrors(form.setError, purchaseErrorFields);

  const submit = form.handleSubmit(async (values) => {
    clearFormError();
    try {
      const purchase = await mutateAsync(toPurchaseInput(values));
      toast({ tone: 'success', title: 'Compra registrada', description: `Factura ${purchase.invoice_number}` });
      void navigate(paths.purchases.detail(purchase.id));
    } catch (error) {
      const line = lineErrorOf(error);
      if (line === undefined) {
        handleError(error);
        return;
      }
      const { code = '', message } = describeError(error);
      form.setError(lineErrorField(code, line), { type: 'server', message }, { shouldFocus: true });
    }
  });

  return { formError, submit };
}
