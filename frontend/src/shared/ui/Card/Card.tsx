import clsx from 'clsx';
import { useId, type ReactNode } from 'react';
import styles from './Card.module.scss';

type Props = {
  /** The card's heading (h2) and the name of its region. */
  title: string;
  /** A link or button beside the title, such as the detail behind the card. */
  action?: ReactNode;
  children: ReactNode;
  className?: string;
};

/**
 * A panel of a screen with its own heading, such as each indicator of the
 * home screen: a region a screen reader can jump to. The action goes under
 * the title on a phone.
 */
export function Card({ title, action, children, className }: Props) {
  const titleId = useId();
  return (
    <section className={clsx(styles.root, className)} aria-labelledby={titleId}>
      <header className={styles.header}>
        <h2 id={titleId} className={styles.title}>
          {title}
        </h2>
        {action}
      </header>
      {children}
    </section>
  );
}
