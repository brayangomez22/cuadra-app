import type { UseFormReturn } from 'react-hook-form';
import { useNavigate } from 'react-router';
import { paths } from '@/app/routes/paths';
import { useApiFormErrors } from '@/shared/forms/useApiFormErrors';
import { useToast } from '@/shared/ui';
import { useSaveCustomer } from '../../../api/customers';
import type { Customer } from '../../../api/types';
import { customerErrorFields, type CustomerValues } from '../../../model/customerForm/customerSchema';
import { toCustomerInput } from '../../../model/customerForm/toCustomerInput';

/** Saves the customer and goes back to the list; an API error goes to its field or stays on the form. */
export function useSaveCustomerForm(form: UseFormReturn<CustomerValues>, current: Customer | undefined) {
  const navigate = useNavigate();
  const { toast } = useToast();
  const { mutateAsync } = useSaveCustomer(current);
  const { formError, handleError, clearFormError } = useApiFormErrors(form.setError, customerErrorFields);

  const submit = form.handleSubmit(async (values) => {
    clearFormError();
    try {
      const saved = await mutateAsync(toCustomerInput(values));
      toast({ tone: 'success', title: current ? 'Cliente guardado' : 'Cliente creado', description: saved.name });
      void navigate(paths.customers.list());
    } catch (error) {
      handleError(error);
    }
  });

  return { formError, submit };
}
