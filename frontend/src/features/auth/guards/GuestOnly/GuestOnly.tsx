import { Navigate, Outlet, useLocation } from 'react-router';
import { redirectTarget } from '../../session/redirect';
import { useAuth } from '../../session/useAuth';
import { SessionLoading } from '../SessionScreen/SessionScreen';

// GuestOnly wraps the login and sign-up: with a session they lead to the app,
// to the page that was asked for before the login if there was one.
export function GuestOnly() {
  const auth = useAuth();
  const location = useLocation();

  if (auth.status === 'loading') return <SessionLoading />;
  if (auth.status === 'authenticated') return <Navigate to={redirectTarget(location.state)} replace />;
  return <Outlet />;
}
