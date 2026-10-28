import { Table, TableBody, TableHead, TableHeaderCell, TableRow } from '@/shared/ui';
import type { Customer, CustomerSegment } from '../../../../api/types';
import { CustomerRow } from './CustomerRow';

type Props = { customers: Customer[]; segments: CustomerSegment[]; canWrite: boolean };

/** The customers of a page: who they are, their document, segment and contact; the name opens the form. */
export function CustomerTable({ customers, segments, canWrite }: Props) {
  const segmentNames = new Map(segments.map((s) => [s.id, s.name]));
  return (
    <Table caption="Clientes">
      <TableHead>
        <TableRow>
          <TableHeaderCell>Cliente</TableHeaderCell>
          <TableHeaderCell>Documento</TableHeaderCell>
          <TableHeaderCell>Segmento</TableHeaderCell>
          <TableHeaderCell>Contacto</TableHeaderCell>
        </TableRow>
      </TableHead>
      <TableBody>
        {customers.map((customer) => (
          <CustomerRow
            key={customer.id}
            customer={customer}
            segment={customer.segment_id && segmentNames.get(customer.segment_id)}
            canWrite={canWrite}
          />
        ))}
      </TableBody>
    </Table>
  );
}
