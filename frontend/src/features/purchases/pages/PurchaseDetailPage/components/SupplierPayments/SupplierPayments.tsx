import { formatDay, formatMoney } from '@/shared/format';
import { QueryState } from '@/shared/query';
import { Spinner, Table, TableBody, TableCell, TableHead, TableHeaderCell, TableNarrow, TableRow } from '@/shared/ui';
import { useSupplierPayments } from '../../../../api/supplierPayments';
import type { Purchase } from '../../../../api/types';
import { paymentMethodLabels } from '../../../../model/purchaseLabels';

/** The payments to the supplier of a purchase on credit, the first one first, with what each left to pay. */
export function SupplierPayments({ purchase }: { purchase: Purchase }) {
  const payments = useSupplierPayments(purchase.id, { enabled: purchase.payment === 'credit' });
  return (
    <QueryState
      queries={[payments]}
      errorTitle="No pudimos cargar los pagos"
      loading={<Spinner label="Cargando los pagos…" />}
      isEmpty={payments.data?.items.length === 0}
      empty={<p>Aún no se le ha pagado nada de esta compra al proveedor.</p>}
    >
      {() => (
        <Table caption="Pagos al proveedor">
          <TableHead>
            <TableRow>
              <TableHeaderCell>Fecha</TableHeaderCell>
              <TableHeaderCell wide>Medio</TableHeaderCell>
              <TableHeaderCell numeric>Valor</TableHeaderCell>
              <TableHeaderCell numeric wide>
                Quedó por pagar
              </TableHeaderCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {payments.data?.items.map((p) => (
              <TableRow key={p.id}>
                <TableCell>
                  {formatDay(p.paid_on)}
                  <TableNarrow>{paymentMethodLabels[p.method]}</TableNarrow>
                </TableCell>
                <TableCell wide>{paymentMethodLabels[p.method]}</TableCell>
                <TableCell numeric>
                  {formatMoney(p.amount)}
                  <TableNarrow>{`Quedó ${formatMoney(p.balance_after)}`}</TableNarrow>
                </TableCell>
                <TableCell numeric wide>
                  {formatMoney(p.balance_after)}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </QueryState>
  );
}
