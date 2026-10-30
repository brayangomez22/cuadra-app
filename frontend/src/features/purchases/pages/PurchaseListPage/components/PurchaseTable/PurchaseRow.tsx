import { paths } from '@/app/routes/paths';
import { formatDay, formatMoney } from '@/shared/format';
import { Badge, TableCell, TableRow, TextLink } from '@/shared/ui';
import type { PurchaseSummary } from '../../../../api/types';
import { paymentLabels, statusLabels } from '../../../../model/purchaseLabels';
import styles from './PurchaseTable.module.scss';

/**
 * A purchase of the list; its invoice number opens it. A purchase in cash has
 * no due date: it was paid. On a phone what is owed and the status go under
 * the total.
 */
export function PurchaseRow({ purchase }: { purchase: PurchaseSummary }) {
  const open = purchase.status === 'open';
  return (
    <TableRow>
      <TableCell>
        <TextLink to={paths.purchases.detail(purchase.id)} className={styles.invoice}>
          {purchase.invoice_number}
        </TextLink>
        <span className={styles.narrow}>{`${purchase.supplier_name} · ${formatDay(purchase.issued_on)}`}</span>
      </TableCell>
      <TableCell className={styles.wide}>{purchase.supplier_name}</TableCell>
      <TableCell className={styles.wide} data-day>
        {formatDay(purchase.issued_on)}
      </TableCell>
      <TableCell className={styles.wide} data-day>
        {purchase.due_on ? formatDay(purchase.due_on) : paymentLabels[purchase.payment]}
      </TableCell>
      <TableCell numeric>
        {formatMoney(purchase.total)}
        <span className={styles.narrow}>
          {open && <span className={styles.owed}>{`Debe ${formatMoney(purchase.balance_due)}`}</span>}
          <Badge tone={open ? 'warning' : 'success'}>{statusLabels[purchase.status]}</Badge>
        </span>
      </TableCell>
      <TableCell numeric className={styles.wide}>
        <span className={styles.balance}>
          {open && formatMoney(purchase.balance_due)}
          <Badge tone={open ? 'warning' : 'success'}>{statusLabels[purchase.status]}</Badge>
        </span>
      </TableCell>
    </TableRow>
  );
}
