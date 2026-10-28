import { Spinner } from '@/shared/ui';
import type { Import } from '../../../../api/types';
import { processed } from '../../../../model/importProgress';
import styles from './ImportProgress.module.scss';

const count = new Intl.NumberFormat('es-CO');

/** The import while the job writes it, batch by batch: it can take a while with thousands of products. */
export function ImportProgress({ found }: { found: Import }) {
  const done = processed(found.counts);
  const total = found.counts.products;
  return (
    <section className={styles.root} aria-live="polite">
      <h2 className={styles.title}>{found.status === 'queued' ? 'En cola…' : 'Importando…'}</h2>
      <progress className={styles.bar} aria-label="Avance de la importación" value={done} max={total} />
      <p className={styles.text}>
        {`${count.format(done)} de ${count.format(total)} productos`}
        <Spinner label="Importando" />
      </p>
      <p className={styles.hint}>Puede seguir usando Cuadra: la importación sigue aunque cierre esta pantalla.</p>
    </section>
  );
}
