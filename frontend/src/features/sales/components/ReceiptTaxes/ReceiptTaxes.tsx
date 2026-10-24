import { formatMoney, formatPercent } from '@/shared/format';
import type { Sale } from '../../api/types';
import styles from './ReceiptTaxes.module.scss';

/** The IVA of the receipt grouped by rate, with its base. */
export function ReceiptTaxes({ taxes }: { taxes: Sale['totals']['taxes'] }) {
  return (
    <table className={styles.root} aria-label="IVA por tasa">
      <thead>
        <tr>
          <th scope="col">Tasa</th>
          <th scope="col">Base</th>
          <th scope="col">IVA</th>
        </tr>
      </thead>
      <tbody>
        {taxes.map((group) => (
          <tr key={group.rate}>
            <td>{formatPercent(group.rate, 'rate')}</td>
            <td>{formatMoney(group.base)}</td>
            <td>{formatMoney(group.tax)}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
