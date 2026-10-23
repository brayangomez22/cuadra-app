import type { ComponentProps } from 'react';
import { Skeleton } from '@/shared/ui';
import type { BusinessTemplate } from '../../../../api/types';
import styles from './BusinessTypeOptions.module.scss';

type RadioProps = Omit<ComponentProps<'input'>, 'type' | 'id'>;

type Props = {
  templates: BusinessTemplate[];
  /** The base of the ids that tie each card to its name and description. */
  id: string;
  labelId: string;
  /** The id of the error message, when there is one. */
  errorId?: string;
  /** react-hook-form's register(), shared by every radio. */
  radio: RadioProps;
};

/** One card per business type, as native radios. */
export function BusinessTypeOptions({ templates, id, labelId, errorId, radio }: Props) {
  return (
    <div
      role="radiogroup"
      aria-labelledby={labelId}
      aria-invalid={errorId ? true : undefined}
      aria-errormessage={errorId}
      className={styles.options}
    >
      {templates.map((template) => (
        <label key={template.id} className={styles.option}>
          <input
            type="radio"
            className={styles.radio}
            value={template.id}
            aria-labelledby={`${id}-${template.id}-name`}
            aria-describedby={`${id}-${template.id}`}
            {...radio}
          />
          <span className={styles.text}>
            <span id={`${id}-${template.id}-name`} className={styles.name}>
              {template.name}
            </span>
            <span id={`${id}-${template.id}`} className={styles.description}>
              {template.description}
            </span>
          </span>
        </label>
      ))}
    </div>
  );
}

/** The place of the cards while the business types load. */
export function BusinessTypeOptionsSkeleton() {
  return (
    <div className={styles.options} aria-busy="true">
      {[0, 1, 2].map((i) => (
        <Skeleton key={i} shape="block" className={styles.placeholder} />
      ))}
    </div>
  );
}
