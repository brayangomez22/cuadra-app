import { useCallback } from 'react';
import { describeError } from '@/shared/api/errors';
import { useToast } from '@/shared/ui';

type ToastContent = { title: string; description?: string };

type Options<TData> = {
  /** The success toast, or none when the screen already shows the result. */
  success?: string | ((data: TData) => ToastContent);
  /** The title of the error toast; its description is the API's message. */
  error: string;
};

/**
 * The success and error toasts of a mutation, to spread into useMutation:
 * `useMutation({ mutationFn, ...useMutationToast({ success: 'Unidad quitada', error: 'No se pudo quitar' }) })`.
 */
export function useMutationToast<TData>({ success, error }: Options<TData>) {
  const { toast } = useToast();

  const onSuccess = useCallback(
    (data: TData) => {
      if (success === undefined) return;
      const content = typeof success === 'string' ? { title: success } : success(data);
      toast({ tone: 'success', ...content });
    },
    [success, toast],
  );

  const onError = useCallback(
    (failure: unknown) => toast({ tone: 'danger', title: error, description: describeError(failure).message }),
    [error, toast],
  );

  return { onSuccess, onError };
}
