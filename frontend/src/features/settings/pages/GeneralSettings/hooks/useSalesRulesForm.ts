import { zodResolver } from '@hookform/resolvers/zod';
import { useForm } from 'react-hook-form';
import { useApiFormErrors } from '@/shared/forms/useApiFormErrors';
import { useToast } from '@/shared/ui';
import { useUpdateSalesRules, type SalesRules } from '../../../api/businessRules';
import {
  salesRulesBody,
  salesRulesInput,
  salesRulesSchema,
  type SalesRulesInput,
  type SalesRulesValues,
} from '../../../model/business/salesRulesSchema';

/** The cashier's discount limit and the required cash session, and their save. */
export function useSalesRulesForm(rules: SalesRules) {
  const form = useForm<SalesRulesInput, unknown, SalesRulesValues>({
    resolver: zodResolver(salesRulesSchema),
    defaultValues: salesRulesInput(rules),
  });
  const { formError, handleError, clearFormError } = useApiFormErrors(form.setError, {
    invalid_discount_limit: 'maxDiscount',
  });
  const save = useUpdateSalesRules();
  const { toast } = useToast();

  async function submit(values: SalesRulesValues) {
    clearFormError();
    try {
      form.reset(salesRulesInput(await save.mutateAsync(salesRulesBody(values))));
      toast({ tone: 'success', title: 'Reglas de venta guardadas' });
    } catch (error) {
      handleError(error);
    }
  }

  return { form, formError, saving: save.isPending, submit: form.handleSubmit(submit) };
}
