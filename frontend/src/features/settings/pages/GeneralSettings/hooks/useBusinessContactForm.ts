import { zodResolver } from '@hookform/resolvers/zod';
import { useForm } from 'react-hook-form';
import { useAuth } from '@/features/auth';
import { useApiFormErrors } from '@/shared/forms/useApiFormErrors';
import { useToast } from '@/shared/ui';
import { useUpdateTenant } from '../../../api/tenant';
import { contactErrorFields, contactSchema, type ContactValues } from '../../../model/business/contactSchema';

/** The business's address, city and phone, from /me, and their save. */
export function useBusinessContactForm() {
  const auth = useAuth();
  const user = auth.status === 'authenticated' ? auth.user : undefined;
  const form = useForm<ContactValues>({
    resolver: zodResolver(contactSchema),
    defaultValues: {
      address: user?.tenant_address ?? '',
      city: user?.tenant_city ?? '',
      phone: user?.tenant_phone ?? '',
    },
  });
  const { formError, handleError, clearFormError } = useApiFormErrors(form.setError, contactErrorFields);
  const save = useUpdateTenant();
  const { toast } = useToast();

  async function submit(values: ContactValues) {
    clearFormError();
    try {
      const tenant = await save.mutateAsync(values);
      form.reset({ address: tenant.address, city: tenant.city, phone: tenant.phone });
      toast({ tone: 'success', title: 'Datos del negocio guardados' });
    } catch (error) {
      handleError(error);
    }
  }

  return { form, formError, saving: save.isPending, submit: form.handleSubmit(submit) };
}
