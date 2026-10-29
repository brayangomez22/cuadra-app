import { formatDateTime, formatDay, formatMoney } from '@/shared/format';
import { TableCell, TableRow } from '@/shared/ui';
import type { CreditMovement } from '../../../../api/types';
import { movementLabel } from '../../../../model/statement/movementLabel';
import styles from './Statement.module.scss';

type Props = { movement: CreditMovement };

/** A line of the statement. A payment and a voided sale lower the balance: their amount shows negative. */
export function StatementRow({ movement }: Props) {
  const lowers = movement.kind !== 'charge';
  return (
    <TableRow>
      <TableCell>{formatDateTime(movement.at)}</TableCell>
      <TableCell>
        {movementLabel(movement)}
        {movement.due_date && <span className={styles.meta}>Vence el {formatDay(movement.due_date)}</span>}
      </TableCell>
      <TableCell numeric>{formatMoney(lowers ? `-${movement.amount}` : movement.amount)}</TableCell>
      <TableCell numeric>{formatMoney(movement.balance)}</TableCell>
    </TableRow>
  );
}
