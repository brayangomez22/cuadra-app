import { useId, type ComponentProps, type ReactNode } from 'react';
import { IconAlert } from '../icons';
import styles from './SegmentedControl.module.scss';

type Option = { value: string; label: string };

type Props = Omit<ComponentProps<'input'>, 'type' | 'value'> & {
  legend: string;
  options: Option[];
  error?: ReactNode;
};

/**
 * A few exclusive choices side by side, all in view: one tap instead of
 * opening a list. Native radios, so keyboard (arrows), forms and register()
 * of react-hook-form work as they are; the rest of the props go to each radio.
 */
export function SegmentedControl({ legend, options, error, ...input }: Props) {
  const errorId = `${useId()}-error`;
  return (
    <fieldset className={styles.root} aria-describedby={error ? errorId : undefined}>
      <legend className={styles.legend}>{legend}</legend>
      <div className={styles.segments}>
        {options.map((option) => (
          <label key={option.value} className={styles.segment}>
            <input
              type="radio"
              className={styles.input}
              value={option.value}
              aria-invalid={error ? true : undefined}
              {...input}
            />
            <span>{option.label}</span>
          </label>
        ))}
      </div>
      {error && (
        <p id={errorId} className={styles.error}>
          <IconAlert className={styles.errorIcon} />
          <span>{error}</span>
        </p>
      )}
    </fieldset>
  );
}
