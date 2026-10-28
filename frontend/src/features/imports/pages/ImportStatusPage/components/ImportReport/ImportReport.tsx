import { paths } from '@/app/routes/paths';
import { ButtonLink, IconSuccess } from '@/shared/ui';
import type { ImportCounts } from '../../../../api/types';
import { resultLines } from '../../../../model/importProgress';
import styles from './ImportReport.module.scss';

/** What the import did: the products created, updated and skipped. */
export function ImportReport({ counts }: { counts: ImportCounts }) {
  return (
    <section className={styles.root}>
      <h2 className={styles.title}>
        <IconSuccess className={styles.icon} />
        Importación terminada
      </h2>
      <ul className={styles.lines} aria-label="Resultado de la importación">
        {resultLines(counts).map((line) => (
          <li key={line}>{line}</li>
        ))}
      </ul>
      <div className={styles.actions}>
        <ButtonLink to={paths.products.list()}>Ver productos</ButtonLink>
        <ButtonLink to={paths.inventory.stock()} variant="secondary">
          Ver inventario
        </ButtonLink>
      </div>
    </section>
  );
}
