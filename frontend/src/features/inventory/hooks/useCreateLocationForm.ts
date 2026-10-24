import { zodResolver } from '@hookform/resolvers/zod';
import { useForm } from 'react-hook-form';
import { z } from 'zod';
import { useApiFormErrors } from '@/shared/forms/useApiFormErrors';
import { useToast } from '@/shared/ui';
import { useCreateLocation } from '../api/locations';
import type { Location } from '../api/types';

const schema = z.object({
  name: z
    .string()
    .trim()
    .min(1, 'Escriba el nombre de la sede')
    .max(100, 'El nombre puede tener máximo 100 caracteres'),
});

const fieldByErrorCode = { location_name_taken: 'name', invalid_location_name: 'name' } as const;

/** The form of a new location; `onDone` runs once the list already has it. */
export function useCreateLocationForm(onDone: (location: Location) => void) {
  const { toast } = useToast();
  const create = useCreateLocation();
  const form = useForm({ resolver: zodResolver(schema), defaultValues: { name: '' } });
  const { formError, handleError, clearFormError } = useApiFormErrors(form.setError, fieldByErrorCode);

  const submit = form.handleSubmit((values) => {
    clearFormError();
    create.mutate(values, {
      onSuccess: (location) => {
        toast({ tone: 'success', title: 'Sede creada', description: location.name });
        onDone(location);
        form.reset();
      },
      onError: handleError,
    });
  });

  function discard() {
    form.reset();
    clearFormError();
  }

  return { form, formError, submit, discard, isPending: create.isPending };
}
