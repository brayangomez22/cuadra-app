import { formatDateTime, formatMoney, todayInBogota } from '@/shared/format';
import {
  EmptyState,
  IconSuccess,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeaderCell,
  TableRow,
} from '@/shared/ui';
import type { Receivable } from '../../../../api/types';
import { DueDate } from './DueDate';

type Props = { receivables: Receivable[] };

/** What the customer owes, sale by sale, from the one due first; a payment goes to them in this order. */
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
          <TableHeaderCell numeric>Valor</TableHeaderCell>
          <TableHeaderCell numeric>Saldo</TableHeaderCell>
          <TableHeaderCell>Vencimiento</TableHeaderCell>
        </TableRow>
      </TableHead>
      <TableBody>
        {receivables.map((receivable) => (
          <TableRow key={receivable.id}>
            <TableCell>{formatDateTime(receivable.created_at)}</TableCell>
            <TableCell numeric>{formatMoney(receivable.amount)}</TableCell>
            <TableCell numeric>{formatMoney(receivable.balance)}</TableCell>
            <TableCell>
              <DueDate dueDate={receivable.due_date} today={today} />
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
