import { zodResolver } from '@hookform/resolvers/zod';
import { useForm } from 'react-hook-form';
import { useNavigate } from 'react-router';
import { paths } from '@/app/routes/paths';
import { useApiFormErrors } from '@/shared/forms/useApiFormErrors';
import { useOpenCashSession } from '../../../api/cashSessions';
import { openCashSchema, type OpenCashInput, type OpenCashValues } from '../../../model/cash/cashSchemas';

// No error of POST /cash-sessions belongs to a field: they go above the form.
const noFields = {};

/**
 * The form that opens the drawer. With a single location it is chosen
 * already; once open, the cashier goes to sell.
 */
export function useOpenCashForm(locations: { value: string }[]) {
  const navigate = useNavigate();
  const open = useOpenCashSession();
  const form = useForm<OpenCashInput, unknown, OpenCashValues>({
    resolver: zodResolver(openCashSchema),
    defaultValues: {
      location_id: locations.length === 1 ? (locations[0]?.value ?? '') : '',
      opening_float: '',
    },
  });
  const { formError, handleError, clearFormError } = useApiFormErrors(form.setError, noFields);

  // mutateAsync, not mutate's callbacks: the open drawer replaces this form
  // once the current session reloads, and its callbacks would not run.
  const submit = form.handleSubmit(async (values) => {
    clearFormError();
    try {
      await open.mutateAsync(values);
    } catch (error) {
      handleError(error);
      return;
    }
    // Opening the drawer is the step before selling.
    void navigate(paths.sales.pos());
  });

  return { form, formError, submit, isPending: open.isPending };
}
