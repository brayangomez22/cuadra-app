import { roleLabels } from '@/features/auth';
import { formatDateTime } from '@/shared/format';
import { Table, TableBody, TableCell, TableHead, TableHeaderCell, TableNarrow, TableRow, TextLink } from '@/shared/ui';
import type { AuditEvent } from '../../../../api/types';
import { actionLabels } from '../../../../model/actions';
import { describeEvent } from '../../../../model/describeEvent';
import { entityLink } from '../../../../model/entityLink';

type Props = { events: AuditEvent[]; userNames: Map<string, string> };

const isRole = (role: string): role is keyof typeof roleLabels => role in roleLabels;
const roleOf = (role: string) => (isRole(role) ? roleLabels[role] : role);

/** Who did what, when, with the figure that matters and where to see it. */
export function AuditTable({ events, userNames }: Props) {
  return (
    <Table caption="Registro de auditoría">
      <TableHead>
        <TableRow>
          <TableHeaderCell>Qué pasó</TableHeaderCell>
          <TableHeaderCell>Quién</TableHeaderCell>
          <TableHeaderCell wide>Cuándo</TableHeaderCell>
          <TableHeaderCell wide>
            <span className="srOnly">Ver</span>
          </TableHeaderCell>
        </TableRow>
      </TableHead>
      <TableBody>
        {events.map((event) => {
          const link = entityLink(event);
          return (
            <TableRow key={event.id}>
              <TableCell>
                {actionLabels[event.action]}
                <TableNarrow>{formatDateTime(event.occurred_at)}</TableNarrow>
                <div>{describeEvent(event)}</div>
              </TableCell>
              <TableCell>
                {userNames.get(event.user_id) ?? 'Usuario borrado'}
                <TableNarrow>{roleOf(event.role)}</TableNarrow>
              </TableCell>
              <TableCell wide>{formatDateTime(event.occurred_at)}</TableCell>
              <TableCell wide>
                {link && (
                  <TextLink to={link.to} tone="quiet">
                    {link.label}
                  </TextLink>
                )}
              </TableCell>
            </TableRow>
          );
        })}
      </TableBody>
    </Table>
  );
}
