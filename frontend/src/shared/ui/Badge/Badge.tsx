import clsx from 'clsx';
import type { ReactNode } from 'react';
import styles from './Badge.module.scss';

type Props = {
  /** The tone adds color; the text must carry the meaning on its own. */
  tone?: 'neutral' | 'success' | 'warning' | 'danger' | 'info';
  icon?: ReactNode;
  children: ReactNode;
  className?: string;
};

export function Badge({ tone = 'neutral', icon, children, className }: Props) {
  return (
    <span className={clsx(styles.root, className)} data-tone={tone}>
      {icon && <span className={styles.icon}>{icon}</span>}
      {children}
    </span>
  );
}
