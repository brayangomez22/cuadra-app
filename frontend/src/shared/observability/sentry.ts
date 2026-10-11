import * as Sentry from '@sentry/react';
import { scrubBreadcrumb, scrubEvent } from './scrub';

// initSentry starts error reporting. Without VITE_SENTRY_DSN (tests, local runs
// without a project) it does nothing. Sentry is for errors only: no tracing and
// no session replay, which would record customers' data on the POS screen.
export function initSentry() {
  const dsn = import.meta.env.VITE_SENTRY_DSN;
  if (!dsn) return;
  Sentry.init({
    dsn,
    environment: import.meta.env.VITE_APP_ENV ?? import.meta.env.MODE,
    release: import.meta.env.VITE_RELEASE,
    sendDefaultPii: false,
    beforeSend: scrubEvent,
    beforeBreadcrumb: scrubBreadcrumb,
  });
}

// recordFailedRequest keeps the X-Request-ID of the last API response that
// failed on the server, so an error reported afterwards leads to that request's
// logs and trace in Grafana.
export function recordFailedRequest(response: Response) {
  if (response.status < 500) return;
  const id = response.headers.get('X-Request-ID');
  if (id) Sentry.setTag('last_failed_request_id', id);
}

// setReportingUser tags the events with the signed-in user's tenant and opaque ID
// (never the email or the name), or clears them when the session ends.
export function setReportingUser(user: { id: string; tenant_id: string } | null) {
  Sentry.setTag('tenant_id', user?.tenant_id);
  Sentry.setUser(user && { id: user.id });
}
