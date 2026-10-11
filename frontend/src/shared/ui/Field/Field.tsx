import clsx from 'clsx';
import { useId, type ReactNode } from 'react';
import { IconAlert } from '../icons';
import { FieldContext } from './FieldContext';
import styles from './Field.module.scss';

type Props = {
  label: ReactNode;
  /** Help text under the control. */
  hint?: ReactNode;
  /** Validation message. Marks the control as aria-invalid and describes it. */
  error?: ReactNode;
  /** One control: Input, Select or another that calls useFieldControl. */
  children: ReactNode;
  className?: string;
};

export function Field({ label, hint, error, children, className }: Props) {
  const id = useId();
  const controlId = `${id}-control`;
  const hintId = hint ? `${id}-hint` : undefined;
  const errorId = error ? `${id}-error` : undefined;
  const describedBy = [hintId, errorId].filter(Boolean).join(' ') || undefined;

  return (
    <div className={clsx(styles.root, className)}>
      <label htmlFor={controlId} className={styles.label}>
        {label}
      </label>
      <FieldContext value={{ controlId, describedBy, invalid: Boolean(error) }}>{children}</FieldContext>
      {hint && (
        <p id={hintId} className={styles.hint}>
          {hint}
        </p>
      )}
      {error && (
        <p id={errorId} className={styles.error}>
          <IconAlert className={styles.errorIcon} />
          <span>{error}</span>
        </p>
      )}
    </div>
  );
}
