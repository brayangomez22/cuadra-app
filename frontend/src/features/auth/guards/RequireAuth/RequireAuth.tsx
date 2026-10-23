import { Navigate, Outlet, useLocation } from 'react-router';
import { paths } from '@/app/routes/paths';
import type { FromState } from '../../session/redirect';
import { useAuth } from '../../session/useAuth';
import { SessionError, SessionLoading } from '../SessionScreen/SessionScreen';

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
    return <Navigate to={paths.login()} replace state={state} />;
  }
  return <Outlet />;
}
