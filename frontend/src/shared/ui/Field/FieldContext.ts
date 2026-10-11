import { createContext, useContext } from 'react';

type FieldContextValue = {
  controlId: string;
  /** Ids of the hint and the error, in reading order. */
  describedBy: string | undefined;
  invalid: boolean;
};

export const FieldContext = createContext<FieldContextValue | null>(null);

type ControlProps = {
  id?: string;
  'aria-describedby'?: string;
  'aria-invalid'?: boolean | 'true' | 'false' | 'grammar' | 'spelling';
};

// useFieldControl wires a control to the Field around it: the label points at
// its id, and the hint and error describe it. Explicit props win. Outside a
// Field it returns the props unchanged.
export function useFieldControl<P extends ControlProps>(props: P): P {
  const field = useContext(FieldContext);
  if (!field) return props;
  const describedBy = [field.describedBy, props['aria-describedby']].filter(Boolean).join(' ') || undefined;
  return {
    ...props,
    id: props.id ?? field.controlId,
    'aria-describedby': describedBy,
    'aria-invalid': props['aria-invalid'] ?? (field.invalid || undefined),
  };
}
