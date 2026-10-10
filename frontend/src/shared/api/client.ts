import createClient from 'openapi-fetch';
import { createAuthFetch } from './authFetch';
import type { paths } from './schema';
import { createSession, type Session } from './session';

// createApiClient builds the typed client for api/openapi.yaml. The paths in the spec
// already carry /api/v1, so baseUrl is only the origin (the page's own by default:
// in development Vite proxies /api to the API).
export function createApiClient(session: Session, baseUrl = window.location.origin) {
  return createClient<paths>({
    baseUrl,
    fetch: createAuthFetch({
      session,
      fetch: (request) => globalThis.fetch(request),
      refreshUrl: new URL('/api/v1/auth/refresh', baseUrl).toString(),
    }),
  });
}

export type ApiClient = ReturnType<typeof createApiClient>;

export const session = createSession();
export const api = createApiClient(session);
