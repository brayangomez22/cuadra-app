import { formatDateTime, formatMoney } from '@/shared/format';
import { Table, TableBody, TableCell, TableHead, TableHeaderCell, TableRow } from '@/shared/ui';
import type { CashMovement } from '../../../../api/types';
import { movementLabels } from '../../../../model/cash/cashLabels';
import styles from './CashMovementsTable.module.scss';

/** The cash ins and outs of the open drawer, outs with their sign. */
export function CashMovementsTable({ movements }: { movements: CashMovement[] }) {
  if (movements.length === 0) return <p className={styles.note}>Todavía no hay ingresos ni egresos.</p>;
  return (
    <Table caption="Ingresos y egresos">
      <TableHead>
        <TableRow>
          <TableHeaderCell>Hora</TableHeaderCell>
          <TableHeaderCell>Tipo</TableHeaderCell>
          <TableHeaderCell>Motivo</TableHeaderCell>
          <TableHeaderCell numeric>Valor</TableHeaderCell>
        </TableRow>
      </TableHead>
      <TableBody>
        {movements.map((m) => (
          <TableRow key={m.id}>
            <TableCell>{formatDateTime(m.created_at)}</TableCell>
            <TableCell>{movementLabels[m.kind]}</TableCell>
            <TableCell>{m.reason}</TableCell>
            <TableCell numeric>{formatMoney(m.kind === 'out' ? `-${m.amount}` : m.amount)}</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
