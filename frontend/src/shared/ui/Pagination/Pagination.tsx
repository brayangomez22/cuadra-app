import clsx from 'clsx';
import { Button } from '../Button';
import styles from './Pagination.module.scss';

type Props = {
  /** Accessible name of the navigation, e.g. "Páginas de productos". */
  label: string;
  /** Results that match, across all pages. */
  total: number;
  limit: number;
  offset: number;
  onOffsetChange: (offset: number) => void;
  className?: string;
};

const count = new Intl.NumberFormat('es-CO');

// Pagination shows which results are on screen ("21–40 de 134") and moves by
// one page. The API pages with limit and offset (api/openapi.yaml).
export function Pagination({ label, total, limit, offset, onOffsetChange, className }: Props) {
  if (total === 0) return null;
  const first = offset + 1;
  const last = Math.min(offset + limit, total);
  const single = offset === 0 && total <= limit;

  return (
    <nav className={clsx(styles.root, className)} aria-label={label}>
      <p className={styles.range} aria-live="polite">
        {count.format(first)}–{count.format(last)} de {count.format(total)}
      </p>
      {!single && (
        <div className={styles.buttons}>
          <Button
            variant="secondary"
            disabled={offset === 0}
            onClick={() => onOffsetChange(Math.max(offset - limit, 0))}
          >
            Anterior
          </Button>
          <Button variant="secondary" disabled={last >= total} onClick={() => onOffsetChange(offset + limit)}>
            Siguiente
          </Button>
        </div>
      )}
    </nav>
  );
}
