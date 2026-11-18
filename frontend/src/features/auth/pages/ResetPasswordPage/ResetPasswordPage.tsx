import { useState } from 'react';
import { useLocation } from 'react-router';
import { paths } from '@/app/routes/paths';
import { Alert, ButtonLink } from '@/shared/ui';
import { AuthPage } from '../../components/AuthPage/AuthPage';
import { LinkProblem } from '../../components/LinkProblem/LinkProblem';
import { tokenFromHash } from '../../model/linkToken';
import { ResetPasswordForm } from './components/ResetPasswordForm/ResetPasswordForm';

/** /restablecer#<token>: the user chooses a new password with the link an owner or admin gave them. */
export function ResetPasswordPage() {
  const token = tokenFromHash(useLocation().hash);
  const [done, setDone] = useState(false);

  if (token === null) return <LinkProblem title="No pudimos abrir el enlace" error={null} />;
  if (done) {
    return (
      <AuthPage title="Contraseña guardada">
        <Alert tone="success">Ya puede ingresar con su contraseña nueva. Sus otras sesiones se cerraron.</Alert>
        <ButtonLink to={paths.login()} size="lg">
          Ingresar
        </ButtonLink>
      </AuthPage>
    );
  }
  return (
    <AuthPage title="Elija una contraseña nueva" description="El enlace sirve una sola vez y vence en 24 horas.">
      <ResetPasswordForm
        token={token}
        onDone={() => {
          setDone(true);
        }}
      />
    </AuthPage>
  );
}
