import { Table, TableBody, TableHead, TableHeaderCell, TableRow } from '@/shared/ui';
import type { PurchaseSummary } from '../../../../api/types';
import { PurchaseRow } from './PurchaseRow';

/** The purchases of a page: the invoice, its supplier, its days, its total and what is still owed. */
export function PurchaseTable({ purchases }: { purchases: PurchaseSummary[] }) {
  return (
    <Table caption="Compras">
      <TableHead>
        <TableRow>
          <TableHeaderCell>Factura</TableHeaderCell>
          <TableHeaderCell>Proveedor</TableHeaderCell>
          <TableHeaderCell>Fecha</TableHeaderCell>
          <TableHeaderCell>Vence</TableHeaderCell>
          <TableHeaderCell numeric>Total</TableHeaderCell>
          <TableHeaderCell numeric>Por pagar</TableHeaderCell>
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
