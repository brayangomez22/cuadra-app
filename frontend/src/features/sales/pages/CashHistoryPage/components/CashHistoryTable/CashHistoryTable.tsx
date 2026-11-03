import { Table, TableBody, TableHead, TableHeaderCell, TableRow } from '@/shared/ui';
import type { CashSessionSummary } from '../../../../api/types';
import { CashHistoryRow } from './CashHistoryRow';

type Props = { sessions: CashSessionSummary[]; locationName: (id: string) => string };

/** One row per drawer: when it opened, who and where, what it sold and how the count came out. */
export function CashHistoryTable({ sessions, locationName }: Props) {
  return (
    <Table caption="Cajas">
      <TableHead>
        <TableRow>
          <TableHeaderCell>Apertura</TableHeaderCell>
          <TableHeaderCell wide>Cajero</TableHeaderCell>
          <TableHeaderCell wide>Sede</TableHeaderCell>
          <TableHeaderCell numeric wide>
            Ventas
          </TableHeaderCell>
          <TableHeaderCell numeric>Total vendido</TableHeaderCell>
          <TableHeaderCell numeric wide>
            Arqueo
          </TableHeaderCell>
        </TableRow>
      </TableHead>
      <TableBody>
        {sessions.map((s) => (
          <CashHistoryRow key={s.id} session={s} location={locationName(s.location_id)} />
        ))}
      </TableBody>
    </Table>
  );
}
