import type { ReactNode } from 'react';
import { Stack } from '@/shared/ui';
import styles from './ReportSection.module.scss';

type Props = {
  title: string;
  description: string;
  /** The export buttons. */
  actions?: ReactNode;
  children: ReactNode;
};

/** A report: its name and what it answers, its export and its filters and figures. */
export function ReportSection({ title, description, actions, children }: Props) {
  return (
    <Stack as="section" gap="4" aria-labelledby="report-title">
      <header className={styles.root}>
        <div className={styles.heading}>
          <h2 id="report-title" className={styles.title}>
            {title}
          </h2>
          <p className={styles.description}>{description}</p>
        </div>
        {actions}
      </header>
      {children}
    </Stack>
  );
}
