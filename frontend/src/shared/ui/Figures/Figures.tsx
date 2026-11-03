import type { ReactNode } from 'react';
import styles from './Figures.module.scss';

/**
 * The key figures of a screen above its detail (what was sold, what is owed),
 * each with its name. One per row on a phone, side by side as the width allows.
 */
export function Figures({ label, children }: { label: string; children: ReactNode }) {
  return (
    <section className={styles.root} aria-label={label}>
      <dl className={styles.figures}>{children}</dl>
    </section>
  );
}

type FigureProps = {
  term: string;
  children: ReactNode;
  /** Only when the figure calls for attention: a zero overdue is not a danger. */
  tone?: 'danger' | 'warning';
};

export function Figure({ term, children, tone }: FigureProps) {
  return (
    <div className={styles.figure} data-tone={tone}>
      <dt>{term}</dt>
      <dd>{children}</dd>
    </div>
  );
}
