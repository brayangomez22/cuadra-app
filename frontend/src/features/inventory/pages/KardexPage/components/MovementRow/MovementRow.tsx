import { formatDateTime, formatMoney, formatQuantity } from '@/shared/format';
import { TableCell, TableRow } from '@/shared/ui';
import type { StockMovement } from '../../../../api/types';
import { movementLabels } from '../../../../model/movementLabels';
import styles from './MovementRow.module.scss';

type Props = { movement: StockMovement; locationName?: string; canSeeCost: boolean };

/** A movement: when, what and where, its signed quantity and the location's balance after it. */
export function MovementRow({ movement, locationName, canSeeCost }: Props) {
  return (
    <TableRow>
      <TableCell className={styles.date}>{formatDateTime(movement.occurred_at)}</TableCell>
      <TableCell>
        {movementLabels[movement.type]}
        <span className={styles.meta}>{locationName}</span>
      </TableCell>
      <TableCell numeric>
        <span data-sign={movement.quantity.startsWith('-') ? 'out' : 'in'} className={styles.quantity}>
          {formatQuantity(movement.quantity, { signed: true })}
        </span>
      </TableCell>
      <TableCell numeric>{formatQuantity(movement.balance_after)}</TableCell>
      {canSeeCost && <TableCell numeric>{movement.unit_cost ? formatMoney(movement.unit_cost) : '—'}</TableCell>}
      <TableCell className={styles.detail}>{movement.reason ?? ''}</TableCell>
    </TableRow>
  );
}
