import type { PostHog } from 'posthog-js';
import { allowedProps } from '../model/allowedProps';
import type { EventName, EventProps } from '../model/catalog';
import { posthogOptions } from './posthogOptions';

/** Who the events are about: the user's pseudonym (GET /me, analytics_id) and the tenant. */
export type AnalyticsSession = { analyticsId: string; tenantId: string };

// The client while a session sends analytics; null otherwise. PostHog is
// loaded on the first session that needs it, so nobody else downloads it.
let client: PostHog | null = null;

/**
 * Starts sending events for session. Without VITE_POSTHOG_KEY (local runs,
 * tests) it does nothing. Called again with another session, it switches.
 */
export async function startAnalytics(session: AnalyticsSession) {
  const key = import.meta.env.VITE_POSTHOG_KEY;
  if (!key) return;
  const posthog = (await import('posthog-js')).default;
  if (!client) posthog.init(key, posthogOptions(import.meta.env.VITE_POSTHOG_HOST));
  posthog.identify(session.analyticsId);
  posthog.group('tenant', session.tenantId);
  client = posthog;
}

/** Stops sending events and forgets the user (logout, consent withdrawn). */
export async function stopAnalytics() {
  await Promise.resolve();
  client?.reset();
  client = null;
}

/**
 * Sends event with the properties its catalog entry allows. Without a
 * session it does nothing, so any screen may call it.
 */
export async function track<E extends EventName>(event: E, props: EventProps<E> = {}) {
  await Promise.resolve();
  client?.capture(event, allowedProps(event, props));
}
