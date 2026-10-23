import { useCallback, useState } from 'react';
import type { FieldValues, Path, UseFormSetError } from 'react-hook-form';
import { describeError } from '@/shared/api/errors';

/** The form fields an API error code belongs to: `{ sku_taken: 'sku' }`. */
export type FieldsByErrorCode<T extends FieldValues> = Partial<Record<string, Path<T> | Path<T>[]>>;

/**
 * Shows a failed request where it belongs: on the fields of its `code`, with
 * the focus on the first one, or as the form's general message when no field
 * applies (including the no-connection message).
 */
export function useApiFormErrors<T extends FieldValues>(setError: UseFormSetError<T>, fields: FieldsByErrorCode<T>) {
  const [formError, setFormError] = useState<string | null>(null);

  const handleError = useCallback(
    (error: unknown) => {
      const { code, message } = describeError(error);
      const target = code === undefined ? undefined : fields[code];
      if (target === undefined) {
        setFormError(message);
        return;
      }
      setFormError(null);
      const names = Array.isArray(target) ? target : [target];
      names.forEach((name, i) => setError(name, { message }, { shouldFocus: i === 0 }));
    },
    [fields, setError],
  );

  const clearFormError = useCallback(() => setFormError(null), []);

  return { formError, handleError, clearFormError };
}
