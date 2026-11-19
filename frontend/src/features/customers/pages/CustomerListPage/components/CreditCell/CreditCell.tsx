import { paths } from '@/app/routes/paths';
import { formatMoney, isZero } from '@/shared/format';
import { Badge, ButtonLink, TableCell } from '@/shared/ui';
import type { Customer } from '../../../../api/types';
import styles from './CreditCell.module.scss';

/**
 * What a customer owes (with what is past due said in words) and the way to
 * their account, in one cell: a column of its own for the link made the
 * table wider than a phone. On a phone the link goes under the amount; from
 * md, beside it.
 */
export function CreditCell({ customer }: { customer: Customer }) {
  const owes = customer.balance && !isZero(customer.balance);
  const overdue = customer.overdue && !isZero(customer.overdue);
  return (
    <TableCell numeric>
      <div className={styles.root}>
        <div className={styles.amounts}>
          {owes ? formatMoney(customer.balance ?? '0') : '—'}
          {overdue && (
            <Badge tone="danger" className={styles.overdue}>{`Vencido ${formatMoney(customer.overdue ?? '0')}`}</Badge>
          )}
        </div>
        <ButtonLink
          variant="ghost"
          to={paths.receivables.account(customer.id)}
          aria-label={`Cuenta de ${customer.name}`}
        >
          Cuenta
        </ButtonLink>
      </div>
    </TableCell>
  );
}
