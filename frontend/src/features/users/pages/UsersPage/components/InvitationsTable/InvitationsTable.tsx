import { roleLabels } from '@/features/auth';
import { formatDateTime } from '@/shared/format';
import {
  Badge,
  Button,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeaderCell,
  TableNarrow,
  TableRow,
} from '@/shared/ui';
import type { Invitation } from '../../../../api/types';

type Props = { invitations: Invitation[]; onRevoke: (invitation: Invitation) => void };

/** The invitations nobody accepted yet: pending or expired, and how to cancel them. */
export function InvitationsTable({ invitations, onRevoke }: Props) {
  return (
    <Table caption="Invitaciones sin aceptar">
      <TableHead>
        <TableRow>
          <TableHeaderCell>Nombre</TableHeaderCell>
          <TableHeaderCell wide>Correo</TableHeaderCell>
          <TableHeaderCell>Rol</TableHeaderCell>
          <TableHeaderCell>Vence</TableHeaderCell>
          <TableHeaderCell>
            <span className="srOnly">Acciones</span>
          </TableHeaderCell>
        </TableRow>
      </TableHead>
      <TableBody>
        {invitations.map((invitation) => (
          <TableRow key={invitation.id}>
            <TableCell>
              {invitation.name}
              <TableNarrow>{invitation.email}</TableNarrow>
            </TableCell>
            <TableCell wide>{invitation.email}</TableCell>
            <TableCell>{roleLabels[invitation.role]}</TableCell>
            <TableCell>
              {invitation.status === 'expired' ? (
                <Badge tone="warning">Vencida</Badge>
              ) : (
                formatDateTime(invitation.expires_at)
              )}
            </TableCell>
            <TableCell>
              <Button variant="ghost" onClick={() => onRevoke(invitation)}>
                Anular
              </Button>
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
