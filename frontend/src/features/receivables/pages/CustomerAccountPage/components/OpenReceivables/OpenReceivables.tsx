import { formatDateTime, formatMoney, todayInBogota } from '@/shared/format';
import {
  EmptyState,
  IconSuccess,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeaderCell,
  TableNarrow,
  TableRow,
} from '@/shared/ui';
import type { Receivable } from '../../../../api/types';
import { DueDate } from './DueDate';

type Props = { receivables: Receivable[] };

/**
 * What the customer owes, sale by sale, from the one due first; a payment goes
 * to them in this order. On a phone the due date goes under the sale, next to
 * what is still owed.
 */
export function OpenReceivables({ receivables }: Props) {
  if (receivables.length === 0) {
    return (
      <EmptyState icon={<IconSuccess />} title="No debe nada" description="No tiene ventas a crédito por pagar." />
    );
  }
  const today = todayInBogota();
  return (
    <Table caption="Cuentas por cobrar abiertas" showCaption>
      <TableHead>
        <TableRow>
          <TableHeaderCell>Venta</TableHeaderCell>
          <TableHeaderCell numeric wide>
            Valor
          </TableHeaderCell>
          <TableHeaderCell numeric>Saldo</TableHeaderCell>
          <TableHeaderCell wide>Vencimiento</TableHeaderCell>
        </TableRow>
      </TableHead>
      <TableBody>
        {receivables.map((receivable) => (
          <TableRow key={receivable.id}>
            <TableCell>
              {formatDateTime(receivable.created_at)}
              <TableNarrow>
                <span>
                  <DueDate dueDate={receivable.due_date} today={today} />
                </span>
              </TableNarrow>
            </TableCell>
            <TableCell numeric wide>
              {formatMoney(receivable.amount)}
            </TableCell>
            <TableCell numeric>{formatMoney(receivable.balance)}</TableCell>
            <TableCell wide>
              <DueDate dueDate={receivable.due_date} today={today} />
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
