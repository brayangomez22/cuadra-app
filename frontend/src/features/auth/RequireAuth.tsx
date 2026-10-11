import { Navigate, Outlet, useLocation } from 'react-router';
import { Alert, Button, Spinner } from '@/shared/ui';
import type { FromState } from './redirect';
import { redirectTarget } from './redirect';
import { useAuth } from './useAuth';
import styles from './RequireAuth.module.scss';

// RequireAuth guards the app routes: without a session it sends the visitor to
// the login, remembering where they wanted to go. Hiding a screen is only for
// the user's sake; the API checks the token and permissions on every request.
export function RequireAuth() {
  const auth = useAuth();
  const location = useLocation();

  if (auth.status === 'loading') return <SessionLoading />;
  if (auth.status === 'error') return <SessionError retry={auth.retry} />;
  if (auth.status === 'anonymous') {
    const state: FromState = { from: { pathname: location.pathname, search: location.search } };
    return <Navigate to="/ingresar" replace state={state} />;
  }
  return <Outlet />;
}

// GuestOnly wraps the login and sign-up: with a session they lead to the app,
// to the page that was asked for before the login if there was one.
export function GuestOnly() {
  const auth = useAuth();
  const location = useLocation();

  if (auth.status === 'loading') return <SessionLoading />;
  if (auth.status === 'authenticated') return <Navigate to={redirectTarget(location.state)} replace />;
  return <Outlet />;
}

function SessionLoading() {
  return (
    <div className={styles.root}>
      <Spinner size="lg" label="Abriendo Cuadra…" />
    </div>
  );
}

function SessionError({ retry }: { retry: () => void }) {
  return (
    <div className={styles.root}>
      <div className={styles.error}>
        <Alert tone="danger" title="No pudimos abrir Cuadra">
          Revise el internet e intente de nuevo. Si sigue igual, puede ser un problema nuestro.
        </Alert>
        <Button onClick={retry}>Intentar de nuevo</Button>
      </div>
    </div>
  );
}
