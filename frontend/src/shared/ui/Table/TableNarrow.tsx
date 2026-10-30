import clsx from 'clsx';
import type { ComponentProps } from 'react';
import styles from './TableNarrow.module.scss';

type Props = ComponentProps<'span'> & {
  /** Until where it shows: md (a phone) or lg, matching the wide columns it stands for. */
  until?: 'md' | 'lg';
};

/** What the wide columns of a row say, under the main value of a cell, until they show. */
export function TableNarrow({ until = 'md', className, ...rest }: Props) {
  return <span className={clsx(styles.root, className)} data-narrow data-until={until} {...rest} />;
}
