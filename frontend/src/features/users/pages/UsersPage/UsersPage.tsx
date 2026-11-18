import { useCurrentUser } from '@/features/auth';
import { QueryState } from '@/shared/query';
import { Button, IconPlus, Page, Stack } from '@/shared/ui';
import { useInvitations, useRevokeInvitation } from '../../api/invitations';
import { useUsers } from '../../api/users';
import { LinkDialog } from '../../components/LinkDialog/LinkDialog';
import { DeactivateDialog } from './components/DeactivateDialog/DeactivateDialog';
import { InvitationsTable } from './components/InvitationsTable/InvitationsTable';
import { InviteDialog } from './components/InviteDialog/InviteDialog';
import { RoleDialog } from './components/RoleDialog/RoleDialog';
import { UsersTable } from './components/UsersTable/UsersTable';
import { useRoleOptions } from './hooks/useRoleOptions';
import { useUserDialogs } from './hooks/useUserDialogs';

/** /usuarios: who works in the company, with which role, and how they join (owner and admin). */
export function UsersPage() {
  const me = useCurrentUser();
  const users = useUsers();
  const invitations = useInvitations();
  const revoke = useRevokeInvitation();
  const roles = useRoleOptions();
  const dialogs = useUserDialogs();

  return (
    <Page
      title="Usuarios"
      description="Invite a su equipo, dele a cada uno su rol y desactive a quien ya no trabaja con usted."
      actions={
        <Button icon={<IconPlus />} onClick={dialogs.openInvite}>
          Invitar usuario
        </Button>
      }
    >
      <Stack gap="6">
        <QueryState queries={[users]} errorTitle="No pudimos cargar los usuarios">
          {() => me && <UsersTable users={users.data ?? []} me={me} {...dialogs.handlers} />}
        </QueryState>
        <QueryState
          queries={[invitations]}
          errorTitle="No pudimos cargar las invitaciones"
          isEmpty={invitations.data?.length === 0}
          empty={null}
        >
          {() => (
            <InvitationsTable
              invitations={invitations.data ?? []}
              onRevoke={(i) => void revoke.mutateAsync(i).catch(() => undefined)}
            />
          )}
        </QueryState>
      </Stack>
      <InviteDialog roles={roles} {...dialogs.invite} />
      <RoleDialog roles={roles} {...dialogs.role} />
      <DeactivateDialog {...dialogs.deactivation} />
      <LinkDialog {...dialogs.link} />
    </Page>
  );
}
