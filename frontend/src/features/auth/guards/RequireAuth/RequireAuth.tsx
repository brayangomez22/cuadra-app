import { useState } from 'react';
import { Navigate, Outlet, useLocation } from 'react-router';
import { paths } from '@/app/routes/paths';
import type { FromState } from '../../session/redirect';
import { useAuth } from '../../session/useAuth';
import { SessionError, SessionLoading } from '../SessionScreen/SessionScreen';

// RequireAuth guards the app routes: without a session it sends the visitor to
// the login, remembering where they wanted to go. Hiding a screen is only for
// the user's sake; the API checks the token and permissions on every request.
// A demo session that ends goes back to the demo, to try another business
// type or role.
export function RequireAuth() {
  const auth = useAuth();
  const location = useLocation();
  const wasDemo = useWasDemo(auth);

  if (auth.status === 'loading') return <SessionLoading />;
  if (auth.status === 'error') return <SessionError retry={auth.retry} />;
  if (auth.status === 'anonymous' && wasDemo) return <Navigate to={paths.demo()} replace />;
  if (auth.status === 'anonymous') {
    const state: FromState = { from: { pathname: location.pathname, search: location.search } };
    return <Navigate to={paths.login()} replace state={state} />;
  }
  return <Outlet />;
}

// useWasDemo remembers whether the last user signed in was a demo one, after
// the session is gone.
function useWasDemo(auth: ReturnType<typeof useAuth>) {
  const [wasDemo, setWasDemo] = useState(false);
  if (auth.status === 'authenticated' && auth.user.demo !== wasDemo) setWasDemo(auth.user.demo);
  return wasDemo;
}
