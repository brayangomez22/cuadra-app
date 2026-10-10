import type { Session } from './session';

type Fetch = (request: Request) => Promise<Response>;

type Options = {
  session: Session;
  // fetch sends the request as is (window.fetch in the app, a fake in tests).
  fetch: Fetch;
  // refreshUrl is the absolute URL of POST /api/v1/auth/refresh.
  refreshUrl: string;
};

// Endpoints under /auth answer 401 for their own reasons (wrong password, invalid
// refresh cookie): refreshing there would loop or hide the error.
const AUTH_PATH = '/api/v1/auth/';

// createAuthFetch returns the fetch used by the API client. It adds the access token
// and, when a request gets a 401, renews the session with the refresh cookie and
// retries that request once. If the renewal fails, it ends the session and returns
// the original 401.
export function createAuthFetch({ session, fetch, refreshUrl }: Options): Fetch {
  // The refresh token rotates and reusing it revokes its whole family, so concurrent
  // 401s must share one refresh instead of sending the same cookie twice.
  let refreshing: Promise<string | null> | null = null;

  // refresh updates the session once per renewal, however many requests wait on it.
  const refresh = () => {
    refreshing ??= renewAccessToken(fetch, refreshUrl)
      .then((token) => {
        if (token) session.setAccessToken(token);
        else session.end();
        return token;
      })
      .finally(() => {
        refreshing = null;
      });
    return refreshing;
  };

  const send = (request: Request, token: string | null) => {
    const authorized = new Request(request);
    if (token) authorized.headers.set('Authorization', `Bearer ${token}`);
    return fetch(authorized);
  };

  return async (request) => {
    // The body can be read only once: keep a copy for the retry.
    const retry = request.clone();
    const response = await send(request, session.getAccessToken());
    if (response.status !== 401 || new URL(request.url).pathname.startsWith(AUTH_PATH)) {
      return response;
    }

    const token = await refresh();
    return token ? send(retry, token) : response;
  };
}

// renewAccessToken exchanges the refresh cookie for a new access token, or returns
// null if the session cannot be renewed.
async function renewAccessToken(fetch: Fetch, refreshUrl: string): Promise<string | null> {
  try {
    const response = await fetch(new Request(refreshUrl, { method: 'POST', credentials: 'same-origin' }));
    if (!response.ok) return null;
    const body: unknown = await response.json();
    return readAccessToken(body);
  } catch {
    return null;
  }
}

function readAccessToken(body: unknown): string | null {
  if (typeof body === 'object' && body !== null && 'access_token' in body) {
    const token = body.access_token;
    if (typeof token === 'string' && token !== '') return token;
  }
  return null;
}
