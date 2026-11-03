import { paths } from '@/app/routes/paths';
import { formatDay, formatMoney, isZero } from '@/shared/format';
import {
  Badge,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeaderCell,
  TableNarrow,
  TableRow,
  TextLink,
} from '@/shared/ui';
import type { SupplierPayable } from '../../../../api/types';

/** One row per supplier owed; its name opens its purchases left to pay. On a phone the split goes under the name. */
export function PayablesTable({ items }: { items: SupplierPayable[] }) {
  return (
    <Table caption="Lo que se debe a cada proveedor">
      <TableHead>
        <TableRow>
          <TableHeaderCell>Proveedor</TableHeaderCell>
          <TableHeaderCell wide>Próximo vencimiento</TableHeaderCell>
          <TableHeaderCell numeric wide>
            Vencido
          </TableHeaderCell>
          <TableHeaderCell numeric wide>
            Vence en 7 días
          </TableHeaderCell>
          <TableHeaderCell numeric>Se debe</TableHeaderCell>
        </TableRow>
      </TableHead>
      <TableBody>
        {items.map((p) => (
          <PayableRow key={p.supplier_id} payable={p} />
        ))}
      </TableBody>
    </Table>
  );
}

function PayableRow({ payable: p }: { payable: SupplierPayable }) {
  const overdue = !isZero(p.overdue);
  const purchases = `${String(p.purchases)} ${p.purchases === 1 ? 'compra' : 'compras'}`;
  return (
    <TableRow>
      <TableCell>
        <TextLink to={paths.purchases.list({ proveedor: p.supplier_id, estado: 'por-pagar' })}>
          {p.supplier_name}
        </TextLink>
        <TableNarrow>{`${purchases} · vence ${formatDay(p.next_due)}`}</TableNarrow>
      </TableCell>
      <TableCell wide>
        {formatDay(p.next_due)} {overdue && <Badge tone="danger">Vencido</Badge>}
      </TableCell>
      <TableCell numeric wide>
        {formatMoney(p.overdue)}
      </TableCell>
      <TableCell numeric wide>
        {formatMoney(p.due_soon)}
      </TableCell>
      <TableCell numeric>
        {formatMoney(p.balance)}
        <TableNarrow>{overdue && <Badge tone="danger">{`Vencido ${formatMoney(p.overdue)}`}</Badge>}</TableNarrow>
      </TableCell>
    </TableRow>
  );
}
