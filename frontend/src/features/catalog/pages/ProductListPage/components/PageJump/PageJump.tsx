import { useState } from 'react';
import { Button, Field, Input } from '@/shared/ui';
import styles from './PageJump.module.scss';

// With fewer pages, Anterior and Siguiente are enough.
const MIN_PAGES = 5;

type Props = { total: number; limit: number; offset: number; onOffsetChange: (offset: number) => void };

/** With thousands of products, go straight to a page instead of clicking Siguiente many times. */
export function PageJump({ total, limit, offset, onOffsetChange }: Props) {
  const pages = Math.ceil(total / limit);
  const [page, setPage] = useState('');
  if (pages < MIN_PAGES) return null;
  const go = () => {
    const number = Math.min(Math.max(Number.parseInt(page, 10) || 1, 1), pages);
    setPage('');
    onOffsetChange((number - 1) * limit);
  };
  return (
    <form
      className={styles.root}
      aria-label="Ir a una página"
      onSubmit={(event) => {
        event.preventDefault();
        go();
      }}
    >
      <Field label={`Página (de 1 a ${String(pages)})`} className={styles.field}>
        <Input
          type="number"
          inputMode="numeric"
          min={1}
          max={pages}
          value={page}
          placeholder={String(offset / limit + 1)}
          onChange={(event) => setPage(event.target.value)}
        />
      </Field>
      <Button type="submit" variant="secondary">
        Ir
      </Button>
    </form>
  );
}
