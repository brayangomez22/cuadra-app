import clsx from 'clsx';
import type { ReactNode } from 'react';
import styles from './EmptyState.module.scss';

type Props = {
  icon?: ReactNode;
  title: ReactNode;
  /** What to do next, in plain words. */
  description?: ReactNode;
  action?: ReactNode;
  /** Heading level that fits the page outline. */
  headingLevel?: 2 | 3 | 4;
  className?: string;
};

export function EmptyState({ icon, title, description, action, headingLevel = 2, className }: Props) {
  const Heading = `h${headingLevel}` as const;
  return (
    <div className={clsx(styles.root, className)}>
      {icon && (
        <span className={styles.icon} aria-hidden="true">
          {icon}
        </span>
      )}
      <Heading className={styles.title}>{title}</Heading>
      {description && <p className={styles.description}>{description}</p>}
      {action && <div className={styles.action}>{action}</div>}
    </div>
  );
}
