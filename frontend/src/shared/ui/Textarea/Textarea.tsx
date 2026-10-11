import clsx from 'clsx';
import type { ComponentProps } from 'react';
import { useFieldControl } from '../Field';
import styles from './Textarea.module.scss';

/** Multi-line text, such as a description. Goes inside a Field like Input. */
export function Textarea({ className, rows = 3, ...props }: ComponentProps<'textarea'>) {
  const rest = useFieldControl(props);
  return <textarea className={clsx(styles.root, className)} rows={rows} {...rest} />;
}
