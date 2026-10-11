import clsx from 'clsx';
import { useId, type ComponentProps, type ReactNode } from 'react';
import { IconCheck } from '../icons';
import styles from './Checkbox.module.scss';

type Props = Omit<ComponentProps<'input'>, 'type'> & {
  label: ReactNode;
  hint?: ReactNode;
};

// A native checkbox: keyboard, forms and screen readers work without Radix.
// The input is invisible on top of the drawn box, so clicks land on it.
export function Checkbox({ label, hint, id, className, ...rest }: Props) {
  const generatedId = useId();
  const inputId = id ?? `${generatedId}-input`;
  const hintId = hint ? `${generatedId}-hint` : undefined;

  return (
    <div className={clsx(styles.root, className)}>
      <span className={styles.control}>
        <input type="checkbox" id={inputId} className={styles.input} aria-describedby={hintId} {...rest} />
        <span className={styles.box} aria-hidden="true">
          <IconCheck className={styles.check} />
        </span>
      </span>
      <span className={styles.text}>
        <label htmlFor={inputId} className={styles.label}>
          {label}
        </label>
        {hint && (
          <span id={hintId} className={styles.hint}>
            {hint}
          </span>
        )}
      </span>
    </div>
  );
}
