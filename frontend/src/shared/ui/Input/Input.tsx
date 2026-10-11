import clsx from 'clsx';
import type { ComponentProps } from 'react';
import { useFieldControl } from '../Field';
import styles from './Input.module.scss';

type Props = ComponentProps<'input'> & {
  /** end for prices and quantities: right-aligned with tabular figures. */
  align?: 'start' | 'end';
};

export function Input({ align = 'start', className, ...props }: Props) {
  const rest = useFieldControl(props);
  return <input className={clsx(styles.root, className)} data-align={align} {...rest} />;
}
