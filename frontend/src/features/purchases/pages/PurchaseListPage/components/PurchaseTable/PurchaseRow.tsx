import { paths } from '@/app/routes/paths';
import { formatDay, formatMoney } from '@/shared/format';
import { Badge, TableCell, TableRow, TextLink } from '@/shared/ui';
import type { PurchaseSummary } from '../../../../api/types';
import { paymentLabels, statusLabels } from '../../../../model/purchaseLabels';
import styles from './PurchaseTable.module.scss';

/** A purchase of the list; its invoice number opens it. A purchase in cash has no due date: it was paid. */
export function PurchaseRow({ purchase }: { purchase: PurchaseSummary }) {
  const open = purchase.status === 'open';
  return (
    <TableRow>
      <TableCell>
        <TextLink to={paths.purchases.detail(purchase.id)}>{purchase.invoice_number}</TextLink>
      </TableCell>
      <TableCell>{purchase.supplier_name}</TableCell>
      <TableCell className={styles.day}>{formatDay(purchase.issued_on)}</TableCell>
      <TableCell className={styles.day}>
        {purchase.due_on ? formatDay(purchase.due_on) : paymentLabels[purchase.payment]}
      </TableCell>
      <TableCell numeric>{formatMoney(purchase.total)}</TableCell>
      <TableCell numeric>
        <span className={styles.balance}>
          {open && formatMoney(purchase.balance_due)}
          <Badge tone={open ? 'warning' : 'success'}>{statusLabels[purchase.status]}</Badge>
        </span>
      </TableCell>
    </TableRow>
  );
}
