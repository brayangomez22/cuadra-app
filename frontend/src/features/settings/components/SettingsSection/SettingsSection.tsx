import { useId, type ReactNode } from 'react';
import styles from './SettingsSection.module.scss';

type Props = {
  title: string;
  description: ReactNode;
  /** The section's main action, such as "Agregar unidad". */
  action?: ReactNode;
  children: ReactNode;
};

/** A section of the settings: its h2, what it is for, its action and its content. */
export function SettingsSection({ title, description, action, children }: Props) {
  const headingId = useId();
  return (
    <section className={styles.root} aria-labelledby={headingId}>
      <header className={styles.header}>
        <div className={styles.heading}>
          <h2 id={headingId} className={styles.title}>
            {title}
          </h2>
          <p className={styles.description}>{description}</p>
        </div>
        {action}
      </header>
      {children}
    </section>
  );
}
