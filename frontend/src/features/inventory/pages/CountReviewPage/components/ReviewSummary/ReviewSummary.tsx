import { useId } from 'react';
import { formatMoney } from '@/shared/format';
import type { StockCountSummary } from '../../../../api/types';
import { accuracyText } from '../../../../model/stockCount/countSummary';
import styles from './ReviewSummary.module.scss';

/** The accuracy of the count, how far it went and the value of its differences (with cost permission). */
export function ReviewSummary({ summary }: { summary: StockCountSummary }) {
  const titleId = useId();
  const net = summary.net_difference_value;
  const absolute = summary.absolute_difference_value;
  return (
    <section className={styles.root} aria-labelledby={titleId}>
      <h2 id={titleId} className={styles.title}>
        Resumen del conteo
      </h2>
      <dl className={styles.figures}>
        <div className={styles.figure}>
          <dt>Exactitud</dt>
          <dd>{accuracyText(summary.accuracy)}</dd>
        </div>
        <div className={styles.figure}>
          <dt>Contados</dt>
          <dd>
            {summary.counted_lines} de {summary.lines}
          </dd>
        </div>
        {net !== undefined && (
          <div className={styles.figure}>
            <dt>Diferencia neta al costo</dt>
            <dd>{formatMoney(net)}</dd>
          </div>
        )}
        {absolute !== undefined && (
          <div className={styles.figure}>
            <dt>Faltantes y sobrantes</dt>
            <dd>{formatMoney(absolute)}</dd>
          </div>
        )}
      </dl>
    </section>
  );
}
