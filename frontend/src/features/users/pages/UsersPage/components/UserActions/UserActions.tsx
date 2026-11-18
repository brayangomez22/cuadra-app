import { Button, Inline } from '@/shared/ui';
import type { User } from '../../../../api/types';
import { userActions } from '../../../../model/userActions';

/** What the page does when a row's button is pressed. */
export type UserHandlers = {
  onChangeRole: (user: User) => void;
  onResetPassword: (user: User) => void;
  onDeactivate: (user: User) => void;
  onActivate: (user: User) => void;
};

type Props = UserHandlers & { user: User; me: { id: string; role: User['role'] } };

/** The buttons of a user's row: only those the manager may use on that user. */
export function UserActions({ user, me, onChangeRole, onResetPassword, onDeactivate, onActivate }: Props) {
  const can = userActions(me, user);
  return (
    <Inline gap="2" wrap justify="end">
      {can.changeRole && (
        <Button variant="secondary" onClick={() => onChangeRole(user)}>
          Cambiar rol
        </Button>
      )}
      {can.resetPassword && (
        <Button variant="ghost" onClick={() => onResetPassword(user)}>
          Enlace de contraseña
        </Button>
      )}
      {can.deactivate && (
        <Button variant="ghost" onClick={() => onDeactivate(user)}>
          Desactivar
        </Button>
      )}
      {can.activate && (
        <Button variant="ghost" onClick={() => onActivate(user)}>
          Reactivar
        </Button>
      )}
    </Inline>
  );
}
