import { useId } from 'react';
import { paths } from '@/app/routes/paths';
import { formatDateTime } from '@/shared/format';
import { Badge, ButtonLink, Inline } from '@/shared/ui';
import type { StockCount } from '../../../../api/types';
import { countedText } from '../../../../model/stockCount/countSummary';
import styles from './OpenCount.module.scss';

type Props = { count: StockCount; canReview: boolean };

/** The count in progress at the location: how far it went, a link to count and, for who manages it, to the differences. */
export function OpenCount({ count, canReview }: Props) {
  const titleId = useId();
  const scope = count.category_ids.length > 0 ? 'Algunas categorías' : 'Toda la sede';
  return (
    <section className={styles.root} aria-labelledby={titleId}>
      <Inline justify="between" align="center" wrap>
        <h2 id={titleId} className={styles.title}>
          Conteo abierto
        </h2>
        <Badge tone="info">{scope}</Badge>
      </Inline>
      <p className={styles.progress}>{countedText(count)}</p>
      <p className={styles.meta}>Iniciado el {formatDateTime(count.started_at)}</p>
      <Inline gap="3" wrap>
        <ButtonLink to={paths.inventory.count(count.id)}>Contar</ButtonLink>
        {canReview && (
          <ButtonLink to={paths.inventory.countReview(count.id)} variant="secondary">
            Ver diferencias
          </ButtonLink>
        )}
      </Inline>
    </section>
  );
}
