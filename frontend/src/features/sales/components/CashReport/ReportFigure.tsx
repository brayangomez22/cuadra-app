import styles from './CashReport.module.scss';

/** A term and its amount in the cash report; strong for the figures of the count. */
export function ReportFigure({ term, value, strong = false }: { term: string; value: string; strong?: boolean }) {
  return (
    <div className={styles.figure} data-strong={strong || undefined}>
      <dt>{term}</dt>
      <dd>{value}</dd>
    </div>
  );
}
