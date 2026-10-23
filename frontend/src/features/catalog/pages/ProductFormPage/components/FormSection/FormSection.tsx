import { useId, type ReactNode } from 'react';
import styles from './FormSection.module.scss';

type Props = { title: string; hint?: ReactNode; children: ReactNode };

/** A section of the product form: a region named by its title. */
export function FormSection({ title, hint, children }: Props) {
  const id = useId();
  return (
    <section className={styles.root} aria-labelledby={id}>
      <h2 id={id} className={styles.title}>
        {title}
      </h2>
      {hint && <p className={styles.hint}>{hint}</p>}
      {children}
    </section>
  );
}

/** The fields of a section: two columns from a tablet on. */
export function FormGrid({ children }: { children: ReactNode }) {
  return <div className={styles.grid}>{children}</div>;
}

/** A field that takes the whole row of the grid. */
export function FormGridWide({ children }: { children: ReactNode }) {
  return <div className={styles.wide}>{children}</div>;
}
