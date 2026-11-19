import { paths } from '@/app/routes/paths';
import { formatMoney, isZero } from '@/shared/format';
import { Badge, ButtonLink, TableCell } from '@/shared/ui';
import type { Customer } from '../../../../api/types';
import styles from './CreditCells.module.scss';

/**
 * What a customer owes (with what is past due said in words) and the way to
 * their account: the cells of who sees the receivables.
 */
export function CreditCells({ customer }: { customer: Customer }) {
  const owes = customer.balance && !isZero(customer.balance);
  const overdue = customer.overdue && !isZero(customer.overdue);
  return (
    <>
      <TableCell numeric>
        {owes ? formatMoney(customer.balance ?? '0') : '—'}
        {overdue && (
          <div>
            <Badge tone="danger">{`Vencido ${formatMoney(customer.overdue ?? '0')}`}</Badge>
          </div>
        )}
      </TableCell>
      <TableCell className={styles.account}>
        <ButtonLink
          variant="ghost"
          to={paths.receivables.account(customer.id)}
          aria-label={`Cuenta de ${customer.name}`}
        >
          Cuenta
        </ButtonLink>
      </TableCell>
    </>
  );
}
