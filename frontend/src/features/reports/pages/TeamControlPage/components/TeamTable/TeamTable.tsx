import { paths } from '@/app/routes/paths';
import { formatMoney, formatMargin } from '@/shared/format';
import { Table, TableBody, TableCell, TableHead, TableHeaderCell, TableRow, TextLink } from '@/shared/ui';
import type { TeamControlReport } from '../../../../api/types';

type Props = {
  users: TeamControlReport['users'];
  /** The period, for the link to each user's audit log; none without audit:read. */
  audit?: { desde: string; hasta: string };
};

// "3 · $ 107.100": how many and for how much.
const countAnd = (count: number, amount: string) => (count === 0 ? '0' : `${String(count)} · ${formatMoney(amount)}`);

/** What each user did in the period, side by side to compare them. */
export function TeamTable({ users, audit }: Props) {
  return (
    <Table caption="Control del equipo por usuario">
      <TableHead>
        <TableRow>
          <TableHeaderCell>Usuario</TableHeaderCell>
          <TableHeaderCell numeric>Anulaciones</TableHeaderCell>
          <TableHeaderCell numeric>Descuentos</TableHeaderCell>
          <TableHeaderCell numeric wide>
            Cambios de precio
          </TableHeaderCell>
          <TableHeaderCell numeric wide>
            Ajustes
          </TableHeaderCell>
          <TableHeaderCell numeric>Faltantes de caja</TableHeaderCell>
        </TableRow>
      </TableHead>
      <TableBody>
        {users.map((user) => (
          <TableRow key={user.user_id}>
            <TableCell>
              {user.name || 'Usuario borrado'}
              {audit && (
                <div>
                  <TextLink to={paths.audit.list({ usuario: user.user_id, ...audit })} tone="quiet">
                    Ver en auditoría
                  </TextLink>
                </div>
              )}
            </TableCell>
            <TableCell numeric>{countAnd(user.voids, user.voids_total)}</TableCell>
            <TableCell numeric>
              {countAnd(user.discounts, user.discounts_amount)}
              {user.discount_rate && <div>{`${formatMargin(user.discount_rate, 'rate')} en promedio`}</div>}
            </TableCell>
            <TableCell numeric wide>
              {user.price_changes}
            </TableCell>
            <TableCell numeric wide>
              {user.adjustments_value === undefined
                ? user.adjustments
                : countAnd(user.adjustments, user.adjustments_value)}
            </TableCell>
            <TableCell numeric>{countAnd(user.cash_differences, user.cash_shortage)}</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
