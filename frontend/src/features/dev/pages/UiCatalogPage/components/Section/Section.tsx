import { useId, type ReactNode } from 'react';
import styles from './Section.module.scss';

/** A group of the catalog, titled with the component's name. */
export function Section({ title, children }: { title: string; children: ReactNode }) {
  const id = useId();
  return (
    <section className={styles.root} aria-labelledby={id}>
      <h2 id={id} className={styles.title}>
        {title}
      </h2>
      {children}
    </section>
  );
}
