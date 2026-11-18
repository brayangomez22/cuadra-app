import { paths } from '@/app/routes/paths';
import type { Invitation, IssuedLink, User } from '../api/types';
import type { IssuedLinkView } from '../components/LinkDialog/LinkDialog';
import { linkUrl } from './linkUrl';

/** The link of a new invitation, to send to whom was invited. */
export function invitationLink(invitation: Invitation, issued: IssuedLink, origin?: string): IssuedLinkView {
  return {
    title: `Invitación de ${invitation.name}`,
    description: `Envíele este enlace a ${invitation.email} para que elija su contraseña.`,
    url: linkUrl(paths.acceptInvitation(), issued.token, origin),
    expiresAt: issued.expires_at,
  };
}

/** The link for a user to choose a new password. */
export function resetLink(user: User, issued: IssuedLink, origin?: string): IssuedLinkView {
  return {
    title: `Enlace de contraseña de ${user.name}`,
    description: 'Envíeselo para que elija una contraseña nueva. Al usarlo se cierran sus sesiones.',
    url: linkUrl(paths.resetPassword(), issued.token, origin),
    expiresAt: issued.expires_at,
  };
}
