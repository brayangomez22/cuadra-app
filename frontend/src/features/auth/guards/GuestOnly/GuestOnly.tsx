import { useState } from 'react';
import { Navigate, Outlet, useLocation } from 'react-router';
import { paths } from '@/app/routes/paths';
import { SessionOpenNotice } from '../../components/SessionOpenNotice/SessionOpenNotice';
import { redirectTarget } from '../../session/redirect';
import { useAuth } from '../../session/useAuth';
import { SessionLoading } from '../SessionScreen/SessionScreen';

// What each page offers to someone who arrives with a session.
function switchLabel(pathname: string): string {
  if (pathname === paths.demo()) return 'Cerrar sesión y elegir otra demo';
  if (pathname === paths.signUp()) return 'Cerrar sesión y registrar otra empresa';
  return 'Cerrar sesión y entrar con otra cuenta';
}

// GuestOnly wraps the login, the sign-up and the demo. Whoever signs in on
// them goes to the app, to the page asked for before the login if there was
// one. Whoever arrives with a session is told so and chooses: back to the app,
// or close it and do what they came for.
export function GuestOnly() {
  const auth = useAuth();
  const location = useLocation();
  // Whether this visit saw the page without a session: then a session is new.
  const [wasGuest, setWasGuest] = useState(false);
  if (auth.status === 'anonymous' && !wasGuest) setWasGuest(true);

  if (auth.status === 'loading') return <SessionLoading />;
  if (auth.status !== 'authenticated') return <Outlet />;
  if (wasGuest) return <Navigate to={redirectTarget(location.state)} replace />;
  return (
    <SessionOpenNotice
      user={auth.user}
      appTarget={paths.home()}
      switchLabel={switchLabel(location.pathname)}
      onLogout={auth.logout}
    />
  );
}
