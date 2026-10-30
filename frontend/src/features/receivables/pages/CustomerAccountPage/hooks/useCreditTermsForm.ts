import { zodResolver } from '@hookform/resolvers/zod';
import { useForm } from 'react-hook-form';
import { useApiFormErrors } from '@/shared/forms/useApiFormErrors';
import { useToast } from '@/shared/ui';
import { useSetCustomerCredit } from '../../../api/credit';
import type { CustomerCredit } from '../../../api/types';
import {
  creditTermsDefaults,
  creditTermsErrorFields,
  creditTermsSchema,
} from '../../../model/creditTerms/creditTermsSchema';

/** The customer's credit limit and term, saved with the API's errors on their fields; onSaved runs after a save. */
export function useCreditTermsForm(credit: CustomerCredit, onSaved: () => void) {
  const { toast } = useToast();
  const save = useSetCustomerCredit(credit.customer_id);
  const form = useForm({ resolver: zodResolver(creditTermsSchema), defaultValues: creditTermsDefaults(credit) });
  const { formError, handleError, clearFormError } = useApiFormErrors(form.setError, creditTermsErrorFields);

  const submit = form.handleSubmit((values) => {
    clearFormError();
    save.mutate(values, {
      onSuccess: (saved) => {
        form.reset(creditTermsDefaults(saved));
        toast({ tone: 'success', title: 'Cupo guardado', description: 'Aplica a las ventas a crédito siguientes.' });
        onSaved();
      },
      onError: handleError,
    });
  });

  return { form, formError, submit, isSaving: save.isPending };
}
