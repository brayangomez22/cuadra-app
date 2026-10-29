import type { UseFormReturn } from 'react-hook-form';
import { useNavigate } from 'react-router';
import { paths } from '@/app/routes/paths';
import { useApiFormErrors } from '@/shared/forms/useApiFormErrors';
import { useToast } from '@/shared/ui';
import { useSaveSupplier } from '../../../api/suppliers';
import type { Supplier } from '../../../api/types';
import {
  supplierErrorFields,
  type SupplierInputValues,
  type SupplierValues,
} from '../../../model/supplierForm/supplierSchema';
import { toSupplierInput } from '../../../model/supplierForm/toSupplierInput';

type SupplierForm = UseFormReturn<SupplierInputValues, unknown, SupplierValues>;

/** Saves the supplier and goes back to the list; an API error goes to its field or stays on the form. */
export function useSaveSupplierForm(form: SupplierForm, current: Supplier | undefined) {
  const navigate = useNavigate();
  const { toast } = useToast();
  const { mutateAsync } = useSaveSupplier(current);
  const { formError, handleError, clearFormError } = useApiFormErrors(form.setError, supplierErrorFields);

  const submit = form.handleSubmit(async (values) => {
    clearFormError();
    try {
      const saved = await mutateAsync(toSupplierInput(values));
      toast({ tone: 'success', title: current ? 'Proveedor guardado' : 'Proveedor creado', description: saved.name });
      void navigate(paths.purchases.suppliers());
    } catch (error) {
      handleError(error);
    }
  });

  return { formError, submit };
}
