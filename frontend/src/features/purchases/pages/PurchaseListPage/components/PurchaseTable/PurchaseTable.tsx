import { Table, TableBody, TableHead, TableHeaderCell, TableRow } from '@/shared/ui';
import type { PurchaseSummary } from '../../../../api/types';
import { PurchaseRow } from './PurchaseRow';

/**
 * The purchases of a page: the invoice, its supplier, its days, its total and
 * what is still owed. On a phone the supplier and the day go under the
 * invoice and what is owed under the total: two columns that fit without
 * scrolling sideways.
 */
export function PurchaseTable({ purchases }: { purchases: PurchaseSummary[] }) {
  return (
    <Table caption="Compras">
      <TableHead>
        <TableRow>
          <TableHeaderCell>Factura</TableHeaderCell>
          <TableHeaderCell wide>Proveedor</TableHeaderCell>
          <TableHeaderCell wide>Fecha</TableHeaderCell>
          <TableHeaderCell wide>Vence</TableHeaderCell>
          <TableHeaderCell numeric>Total</TableHeaderCell>
          <TableHeaderCell numeric wide>
            Por pagar
          </TableHeaderCell>
        </TableRow>
      </TableHead>
      <TableBody>
        {purchases.map((purchase) => (
          <PurchaseRow key={purchase.id} purchase={purchase} />
        ))}
      </TableBody>
    </Table>
  );
}
