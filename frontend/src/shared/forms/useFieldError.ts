import { get, useFormState, type Control, type FieldError, type FieldValues, type Path } from 'react-hook-form';

/**
 * The validation message of a field by its path ("variants.2.sku"), for
 * Field's error. With name, it re-renders only when that part of the form's
 * errors changes.
 */
export function useFieldError<T extends FieldValues>(control: Control<T>, name?: Path<T>) {
  const { errors } = useFormState({ control, ...(name && { name }) });
  return (path: Path<T>) => (get(errors, path) as FieldError | undefined)?.message;
}
