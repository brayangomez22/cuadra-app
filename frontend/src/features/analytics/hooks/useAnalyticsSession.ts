import { useEffect } from 'react';
import { onErrorShown } from '@/shared/observability';
import { cleanUrl } from '../model/cleanUrl';
import { startAnalytics, stopAnalytics, track } from '../tracker/tracker';

/** The part of GET /me the analytics needs. */
type AnalyticsUser = { analytics_id?: string; tenant_id: string } | null;

/**
 * Sends analytics while user brings an analytics_id (the tenant consented,
 * the "analytics" flag is on and the backend is configured), and stops when
 * it does not: logout, consent withdrawn or flag turned off.
 */
export function useAnalyticsSession(user: AnalyticsUser) {
  const analyticsId = user?.analytics_id;
  const tenantId = user?.tenant_id;
  useEffect(() => {
    if (!analyticsId || !tenantId) {
      void stopAnalytics();
      return;
    }
    void startAnalytics({ analyticsId, tenantId });
    // The errors the app shows, by their code: what fails most for the users.
    return onErrorShown((code) => void track('error_shown', { code, screen: cleanUrl(window.location.pathname) }));
  }, [analyticsId, tenantId]);
}
