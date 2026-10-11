import clsx from 'clsx';
import type { ReactNode } from 'react';
import styles from './Page.module.scss';

type Props = {
  /** The h1 of the screen. */
  title: ReactNode;
  description?: ReactNode;
  /** Main actions of the screen, after the title (the primary one first). */
  actions?: ReactNode;
  children?: ReactNode;
  className?: string;
};

// Page is the frame of a screen inside the app layout: its title, actions and
// content, centered up to --size-page on wide screens.
export function Page({ title, description, actions, children, className }: Props) {
  return (
    <div className={clsx(styles.root, className)}>
      <header className={styles.header}>
        <div className={styles.heading}>
          <h1 className={styles.title}>{title}</h1>
          {description && <p className={styles.description}>{description}</p>}
        </div>
        {actions && <div className={styles.actions}>{actions}</div>}
      </header>
      {children}
    </div>
  );
}
