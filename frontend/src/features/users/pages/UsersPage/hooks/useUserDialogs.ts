import { useState } from 'react';
import { useIssuePasswordReset, useSetUserActive } from '../../../api/users';
import type { Invitation, IssuedLink, User } from '../../../api/types';
import type { IssuedLinkView } from '../../../components/LinkDialog/LinkDialog';
import { invitationLink, resetLink } from '../../../model/issuedLinks';

// The mutations' errors are shown by their toasts: the promise is only awaited here.
const ignore = () => undefined;

/** The dialogs of the users screen and what their buttons do. */
export function useUserDialogs() {
  const [inviting, setInviting] = useState(false);
  const [roleOf, setRoleOf] = useState<User | null>(null);
  const [toDeactivate, setToDeactivate] = useState<User | null>(null);
  const [link, setLink] = useState<IssuedLinkView | null>(null);
  const setActive = useSetUserActive();
  const issueReset = useIssuePasswordReset();

  const deactivate = async (user: User) => {
    await setActive.mutateAsync({ user, active: false });
    setToDeactivate(null);
  };

  return {
    invite: {
      open: inviting,
      onClose: () => setInviting(false),
      onInvited: (created: { invitation: Invitation; link: IssuedLink }) =>
        setLink(invitationLink(created.invitation, created.link)),
    },
    openInvite: () => setInviting(true),
    role: { user: roleOf, onClose: () => setRoleOf(null) },
    deactivation: {
      user: toDeactivate,
      deactivating: setActive.isPending,
      onConfirm: (user: User) => void deactivate(user).catch(ignore),
      onClose: () => setToDeactivate(null),
    },
    link: { link, onClose: () => setLink(null) },
    handlers: {
      onChangeRole: setRoleOf,
      onDeactivate: setToDeactivate,
      onActivate: (user: User) => void setActive.mutateAsync({ user, active: true }).catch(ignore),
      onResetPassword: (user: User) =>
        void issueReset.mutateAsync(user).then((issued) => setLink(resetLink(user, issued)), ignore),
    },
  };
}
