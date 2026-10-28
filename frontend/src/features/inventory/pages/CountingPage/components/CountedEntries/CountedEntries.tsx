import { useId } from 'react';
import { useCan } from '@/features/auth';
import { QueryState } from '@/shared/query';
import { Spinner } from '@/shared/ui';
import { useCountEntries } from '../../../../api/countEntries';
import { useRowProducts } from '../../../../hooks/useRowProducts';
import { EntryItem } from './EntryItem';
import styles from './CountedEntries.module.scss';

type Props = { countId: string; open: boolean };

/** The latest entries: the user's own, or everybody's for who manages counts. They can be deleted while it is open. */
export function CountedEntries({ countId, open }: Props) {
  const titleId = useId();
  const entries = useCountEntries(countId);
  const items = entries.data?.items ?? [];
  const { products, productsById } = useRowProducts(items, false);
  const title = useCan()('inventory:count_manage') ? 'Lo contado en este conteo' : 'Lo que usted contó';

  return (
    <section className={styles.root} aria-labelledby={titleId}>
      <h2 id={titleId} className={styles.title}>
        {title}
      </h2>
      <QueryState
        queries={[entries]}
        errorTitle="No pudimos cargar lo contado"
        loading={<Spinner label="Cargando lo contado…" />}
        isEmpty={items.length === 0}
        empty={<p className={styles.empty}>Aún no ha registrado nada.</p>}
      >
        {() => (
          <ul className={styles.list} aria-label={title}>
            {items.map((entry) => (
              <EntryItem
                key={entry.id}
                countId={countId}
                entry={entry}
                product={productsById.get(entry.product_id)}
                loading={products.isPending}
                canDelete={open}
              />
            ))}
          </ul>
        )}
      </QueryState>
    </section>
  );
}
