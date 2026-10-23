import { useEffect, type ReactNode } from 'react';
import { setReportingUser } from '@/shared/observability';
import { useClearCacheOnSessionEnd } from '../api/session';
import { useAuth } from './useAuth';

// AuthProvider reacts to the end of the session (logout, or a refresh that
// failed): the cache of the previous user goes away and there is no user, so
// RequireAuth sends the page to the login.
//
// It also tags error reports with the signed-in user's tenant and opaque ID.
export function AuthProvider({ children }: { children: ReactNode }) {
  const auth = useAuth();
  const userId = auth.status === 'authenticated' ? auth.user.id : null;
  const tenantId = auth.status === 'authenticated' ? auth.user.tenant_id : null;

  useEffect(() => {
    setReportingUser(userId && tenantId ? { id: userId, tenant_id: tenantId } : null);
  }, [userId, tenantId]);

  useClearCacheOnSessionEnd();

  return children;
}
