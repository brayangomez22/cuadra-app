import type { ReactNode } from 'react';
import styles from './FormGrid.module.scss';

/** Sample fields in one column on a phone and two from md up. */
export function FormGrid({ children }: { children: ReactNode }) {
  return <div className={styles.root}>{children}</div>;
}
