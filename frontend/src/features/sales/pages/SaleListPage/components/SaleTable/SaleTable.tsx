import { Table, TableBody, TableHead, TableHeaderCell, TableRow } from '@/shared/ui';
import type { SaleSummary } from '../../../../api/types';
import { SaleRow } from './SaleRow';

type Props = { sales: SaleSummary[]; locationName: (id: string) => string };

/** The sales of a page. On a phone the time and location go under the number and the status under the total: two columns. */
export function SaleTable({ sales, locationName }: Props) {
  return (
    <Table caption="Ventas">
      <TableHead>
        <TableRow>
          <TableHeaderCell>N.º</TableHeaderCell>
          <TableHeaderCell wide>Hora</TableHeaderCell>
          <TableHeaderCell wide>Sede</TableHeaderCell>
          <TableHeaderCell numeric>Total</TableHeaderCell>
          <TableHeaderCell wide>Estado</TableHeaderCell>
        </TableRow>
      </TableHead>
      <TableBody>
        {sales.map((sale) => (
          <SaleRow key={sale.id} sale={sale} location={locationName(sale.location_id)} />
        ))}
      </TableBody>
    </Table>
  );
}
