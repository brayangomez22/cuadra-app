import { formatDateTime, formatMoney, formatQuantity } from '@/shared/format';
import { TableCell, TableNarrow, TableRow } from '@/shared/ui';
import type { StockMovement } from '../../../../api/types';
import { MovementLabel } from './MovementLabel';
import styles from './MovementRow.module.scss';

type Props = {
  movement: StockMovement;
  /** The movement's location, when the kardex shows every location: its balance is that location's. */
  locationName?: string;
  canSeeCost: boolean;
  withDetail: boolean;
};

/**
 * A movement: when and what, its signed quantity and the balance its location
 * kept. On a phone the day goes under the movement, so the quantity and the
 * balance keep their columns.
 */
export function MovementRow({ movement, locationName, canSeeCost, withDetail }: Props) {
  return (
    <TableRow>
      <TableCell wide className={styles.date}>
        {formatDateTime(movement.occurred_at)}
      </TableCell>
      <TableCell>
        <MovementLabel movement={movement} />
        <TableNarrow>{formatDateTime(movement.occurred_at)}</TableNarrow>
      </TableCell>
      <TableCell numeric>
        <span data-sign={movement.quantity.startsWith('-') ? 'out' : 'in'} className={styles.quantity}>
          {formatQuantity(movement.quantity, { signed: true })}
        </span>
      </TableCell>
      <TableCell numeric>
        {formatQuantity(movement.balance_after)}
        {locationName && <span className={styles.location}>en {locationName}</span>}
      </TableCell>
      {canSeeCost && (
        <TableCell numeric wide>
          {movement.unit_cost ? formatMoney(movement.unit_cost) : '—'}
        </TableCell>
      )}
      {withDetail && (
        <TableCell wide className={styles.detail}>
          {movement.reason ?? ''}
        </TableCell>
      )}
    </TableRow>
  );
}
