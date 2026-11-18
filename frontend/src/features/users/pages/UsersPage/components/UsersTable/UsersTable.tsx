import { roleLabels } from '@/features/auth';
import { Badge, Table, TableBody, TableCell, TableHead, TableHeaderCell, TableNarrow, TableRow } from '@/shared/ui';
import type { User } from '../../../../api/types';
import { UserActions, type UserHandlers } from '../UserActions/UserActions';

type Props = UserHandlers & { users: User[]; me: { id: string; role: User['role'] } };

/** The company's users: who, with which role, whether they can enter, and what the manager may do. */
export function UsersTable({ users, me, ...handlers }: Props) {
  return (
    <Table caption="Usuarios de la empresa">
      <TableHead>
        <TableRow>
          <TableHeaderCell>Nombre</TableHeaderCell>
          <TableHeaderCell wide>Correo</TableHeaderCell>
          <TableHeaderCell>Rol</TableHeaderCell>
          <TableHeaderCell>Estado</TableHeaderCell>
          <TableHeaderCell>
            <span className="srOnly">Acciones</span>
          </TableHeaderCell>
        </TableRow>
      </TableHead>
      <TableBody>
        {users.map((user) => (
          <TableRow key={user.id}>
            <TableCell>
              {user.name}
              <TableNarrow>{user.email}</TableNarrow>
            </TableCell>
            <TableCell wide>{user.email}</TableCell>
            <TableCell>{roleLabels[user.role]}</TableCell>
            <TableCell>
              <Badge tone={user.active ? 'success' : 'neutral'}>{user.active ? 'Activo' : 'Inactivo'}</Badge>
            </TableCell>
            <TableCell>
              <UserActions user={user} me={me} {...handlers} />
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
