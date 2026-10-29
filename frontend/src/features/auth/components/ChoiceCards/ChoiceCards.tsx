import type { ComponentProps } from 'react';
import { Skeleton } from '@/shared/ui';
import styles from './ChoiceCards.module.scss';

type RadioProps = Omit<ComponentProps<'input'>, 'type' | 'id'>;

/** One card: what is chosen, its name and, if it helps, what it means. */
export type Choice = { value: string; label: string; description?: string };

type Props = {
  options: Choice[];
  /** The base of the ids that tie each card to its name and description. */
  id: string;
  labelId: string;
  /** The id of the error message, when there is one. */
  errorId?: string;
  /** react-hook-form's register(), shared by every radio. */
  radio: RadioProps;
};

/** One card per option, as native radios (the business type, the demo's role). */
export function ChoiceCards({ options, id, labelId, errorId, radio }: Props) {
  return (
    <div
      role="radiogroup"
      aria-labelledby={labelId}
      aria-invalid={errorId ? true : undefined}
      aria-errormessage={errorId}
      className={styles.root}
    >
      {options.map((option) => (
        <label key={option.value} className={styles.option}>
          <input
            type="radio"
            className={styles.radio}
            value={option.value}
            aria-labelledby={`${id}-${option.value}-name`}
            aria-describedby={option.description ? `${id}-${option.value}` : undefined}
            {...radio}
          />
          <span className={styles.text}>
            <span id={`${id}-${option.value}-name`} className={styles.name}>
              {option.label}
            </span>
            {option.description && (
              <span id={`${id}-${option.value}`} className={styles.description}>
                {option.description}
              </span>
            )}
          </span>
        </label>
      ))}
    </div>
  );
}

/** The place of the cards while the options load. */
export function ChoiceCardsSkeleton() {
  return (
    <div className={styles.root} aria-busy="true">
      {[0, 1, 2].map((i) => (
        <Skeleton key={i} shape="block" className={styles.placeholder} />
      ))}
    </div>
  );
}
