import { formatDateTime, formatDay, formatMoney } from '@/shared/format';
import { TableCell, TableNarrow, TableRow } from '@/shared/ui';
import type { CreditMovement } from '../../../../api/types';
import { movementLabel } from '../../../../model/statement/movementLabel';
import styles from './Statement.module.scss';

type Props = { movement: CreditMovement };

/**
 * A line of the statement. A payment and a voided sale lower the balance:
 * their amount shows negative. On a phone the day and the amount go under the
 * movement, so the balance it left keeps its column.
 */
export function StatementRow({ movement }: Props) {
  const amount = formatMoney(movement.kind === 'charge' ? movement.amount : `-${movement.amount}`);
  return (
    <TableRow>
      <TableCell wide>{formatDateTime(movement.at)}</TableCell>
      <TableCell>
        {movementLabel(movement)}
        {movement.due_date && <span className={styles.meta}>Vence el {formatDay(movement.due_date)}</span>}
        <TableNarrow>{`${formatDateTime(movement.at)} · ${amount}`}</TableNarrow>
      </TableCell>
      <TableCell numeric wide>
        {amount}
      </TableCell>
      <TableCell numeric>{formatMoney(movement.balance)}</TableCell>
    </TableRow>
  );
}
