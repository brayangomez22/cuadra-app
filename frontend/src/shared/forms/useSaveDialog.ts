import type { FieldValues, UseFormReturn } from 'react-hook-form';
import { useApiFormErrors, type FieldsByErrorCode } from '@/shared/forms/useApiFormErrors';
import { useToast } from '@/shared/ui';

type Options<T extends FieldValues, TSaved> = {
  /** The field each 4xx code of the endpoint belongs to (api/openapi.yaml). */
  fields: FieldsByErrorCode<T>;
  save: (values: T) => Promise<TSaved>;
  /** The success toast: "Unidad guardada", with what was saved. */
  success: (saved: TSaved) => { title: string; description?: string };
  onClose: () => void;
};

/**
 * Saves the form of a settings dialog: on success it shows a toast and closes;
 * an API error goes to its field or stays as the form's message.
 */
export function useSaveDialog<T extends FieldValues, TSaved>(
  form: Pick<UseFormReturn<T, unknown, T>, 'handleSubmit' | 'setError'>,
  { fields, save, success, onClose }: Options<T, TSaved>,
) {
  const { toast } = useToast();
  const { formError, handleError, clearFormError } = useApiFormErrors(form.setError, fields);

  const onSubmit = form.handleSubmit(async (values) => {
    clearFormError();
    try {
      const saved = await save(values);
      toast({ tone: 'success', ...success(saved) });
      onClose();
    } catch (error) {
      handleError(error);
    }
  });

  const close = () => {
    clearFormError();
    onClose();
  };

  return { formError, onSubmit, close };
}
