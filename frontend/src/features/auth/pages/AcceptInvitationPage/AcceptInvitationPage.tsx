import { useLocation } from 'react-router';
import { roleLabels } from '../../model/roles';
import { Spinner } from '@/shared/ui';
import { useInvitationPreview } from '../../api/links';
import { AuthPage } from '../../components/AuthPage/AuthPage';
import { LinkProblem } from '../../components/LinkProblem/LinkProblem';
import { tokenFromHash } from '../../model/linkToken';
import { AcceptInvitationForm } from './components/AcceptInvitationForm/AcceptInvitationForm';

/** /invitacion#<token>: whoever was invited chooses a password and enters the company. */
export function AcceptInvitationPage() {
  const token = tokenFromHash(useLocation().hash);
  const preview = useInvitationPreview(token);

  if (token === null || preview.isError) {
    return <LinkProblem title="No pudimos abrir la invitación" error={token === null ? null : preview.error} />;
  }
  if (!preview.data) return <Spinner label="Cargando la invitación…" />;
  const invitation = preview.data;
  return (
    <AuthPage
      title={`Únase a ${invitation.tenant_name}`}
      description={`Lo invitaron como ${roleLabels[invitation.role]} con el correo ${invitation.email}. Elija su contraseña para entrar.`}
    >
      <AcceptInvitationForm token={token} defaultName={invitation.name} />
    </AuthPage>
  );
}
