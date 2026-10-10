import { createAuthFetch } from './authFetch';
import { createSession, type Session } from './session';

const BASE = 'http://localhost';
const REFRESH_PATH = '/api/v1/auth/refresh';

type Handler = (request: Request) => Response | Promise<Response>;

// fakeApi is an in-memory fetch: one handler per path, and it records every request it receives.
function fakeApi(handlers: Record<string, Handler>) {
  const calls: { path: string; authorization: string | null; body: string }[] = [];
  const fetch = async (request: Request): Promise<Response> => {
    const path = new URL(request.url).pathname;
    calls.push({ path, authorization: request.headers.get('Authorization'), body: await request.clone().text() });
    const handler = handlers[path];
    if (!handler) throw new Error(`unexpected request to ${path}`);
    return handler(request);
  };
  return { fetch, calls, callsTo: (path: string) => calls.filter((c) => c.path === path) };
}

const json = (status: number, body: unknown) =>
  new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } });

const unauthorized = () => json(401, { error: { code: 'unauthorized', message: 'Sesión no válida' } });

const refreshed = (token: string) =>
  json(200, { access_token: token, token_type: 'Bearer', expires_in: 900, user: {} });

// requiresToken answers 200 only to the given access token, like a protected endpoint.
const requiresToken =
  (token: string): Handler =>
  (request) =>
    request.headers.get('Authorization') === `Bearer ${token}` ? json(200, { ok: true }) : unauthorized();

function setup(handlers: Record<string, Handler>, initialToken: string | null = 'old') {
  const api = fakeApi(handlers);
  const session: Session = createSession();
  if (initialToken) session.setAccessToken(initialToken);
  const onEnd = vi.fn();
  session.onEnd(onEnd);
  const authFetch = createAuthFetch({ session, fetch: api.fetch, refreshUrl: `${BASE}${REFRESH_PATH}` });
  return { api, session, onEnd, authFetch };
}

describe('authFetch', () => {
  it('agrega el access token en el header Authorization', async () => {
    const { api, authFetch } = setup({ '/api/v1/me': requiresToken('old') });

    const response = await authFetch(new Request(`${BASE}/api/v1/me`));

    expect(response.status).toBe(200);
    expect(api.calls[0]?.authorization).toBe('Bearer old');
  });

  it('no agrega Authorization si no hay sesión', async () => {
    const { api, authFetch } = setup({ '/api/v1/auth/login': () => json(200, {}) }, null);

    await authFetch(new Request(`${BASE}/api/v1/auth/login`, { method: 'POST' }));

    expect(api.calls[0]?.authorization).toBeNull();
  });

  it('ante un 401 refresca el token y reintenta una vez con el token nuevo', async () => {
    const { api, session, authFetch } = setup({
      '/api/v1/me': requiresToken('new'),
      [REFRESH_PATH]: () => refreshed('new'),
    });

    const response = await authFetch(new Request(`${BASE}/api/v1/me`));

    expect(response.status).toBe(200);
    expect(api.calls.map((c) => [c.path, c.authorization])).toEqual([
      ['/api/v1/me', 'Bearer old'],
      [REFRESH_PATH, null],
      ['/api/v1/me', 'Bearer new'],
    ]);
    expect(session.getAccessToken()).toBe('new');
  });

  it('reenvía el body original en el reintento', async () => {
    const { api, authFetch } = setup({
      '/api/v1/products': requiresToken('new'),
      [REFRESH_PATH]: () => refreshed('new'),
    });
    const body = JSON.stringify({ sku: 'CEM-50', name: 'Cemento gris 50kg' });

    await authFetch(new Request(`${BASE}/api/v1/products`, { method: 'POST', body }));

    expect(api.callsTo('/api/v1/products').map((c) => c.body)).toEqual([body, body]);
  });

  it('si el refresh falla, cierra la sesión y devuelve el 401', async () => {
    const { api, session, onEnd, authFetch } = setup({
      '/api/v1/me': requiresToken('new'),
      [REFRESH_PATH]: () => json(401, { error: { code: 'invalid_session', message: 'Sesión no válida' } }),
    });

    const response = await authFetch(new Request(`${BASE}/api/v1/me`));

    expect(response.status).toBe(401);
    expect(api.callsTo('/api/v1/me')).toHaveLength(1);
    expect(session.getAccessToken()).toBeNull();
    expect(onEnd).toHaveBeenCalledOnce();
  });

  it('sin token en memoria (tras recargar la página) refresca con la cookie y reintenta', async () => {
    const { api, authFetch } = setup(
      { '/api/v1/me': requiresToken('new'), [REFRESH_PATH]: () => refreshed('new') },
      null,
    );

    const response = await authFetch(new Request(`${BASE}/api/v1/me`));

    expect(response.status).toBe(200);
    expect(api.callsTo('/api/v1/me').map((c) => c.authorization)).toEqual([null, 'Bearer new']);
  });

  it('sin token en memoria, si el refresh falla igual cierra la sesión', async () => {
    const { onEnd, authFetch } = setup(
      { '/api/v1/me': unauthorized, [REFRESH_PATH]: unauthorized },
      null,
    );

    await authFetch(new Request(`${BASE}/api/v1/me`));

    expect(onEnd).toHaveBeenCalledOnce();
  });

  it('si varias requests esperan un mismo refresh fallido, cierra la sesión una sola vez', async () => {
    const { onEnd, authFetch } = setup({
      '/api/v1/me': unauthorized,
      '/api/v1/products': unauthorized,
      [REFRESH_PATH]: unauthorized,
    });

    await Promise.all([
      authFetch(new Request(`${BASE}/api/v1/me`)),
      authFetch(new Request(`${BASE}/api/v1/products`)),
    ]);

    expect(onEnd).toHaveBeenCalledOnce();
  });

  it('si el refresh no responde (error de red), cierra la sesión y devuelve el 401', async () => {
    const { session, onEnd, authFetch } = setup({
      '/api/v1/me': requiresToken('new'),
      [REFRESH_PATH]: () => Promise.reject(new TypeError('Failed to fetch')),
    });

    const response = await authFetch(new Request(`${BASE}/api/v1/me`));

    expect(response.status).toBe(401);
    expect(session.getAccessToken()).toBeNull();
    expect(onEnd).toHaveBeenCalledOnce();
  });

  it('si el reintento también responde 401 no vuelve a refrescar', async () => {
    const { api, authFetch } = setup({
      '/api/v1/me': unauthorized,
      [REFRESH_PATH]: () => refreshed('new'),
    });

    const response = await authFetch(new Request(`${BASE}/api/v1/me`));

    expect(response.status).toBe(401);
    expect(api.callsTo(REFRESH_PATH)).toHaveLength(1);
    expect(api.callsTo('/api/v1/me')).toHaveLength(2);
  });

  it('varias requests con 401 simultáneas comparten un solo refresh', async () => {
    let releaseRefresh!: () => void;
    const refreshGate = new Promise<void>((resolve) => (releaseRefresh = resolve));
    const { api, authFetch } = setup({
      '/api/v1/me': requiresToken('new'),
      '/api/v1/products': requiresToken('new'),
      [REFRESH_PATH]: async () => {
        await refreshGate;
        return refreshed('new');
      },
    });

    const pending = Promise.all([
      authFetch(new Request(`${BASE}/api/v1/me`)),
      authFetch(new Request(`${BASE}/api/v1/products`)),
    ]);
    await vi.waitFor(() => expect(api.callsTo(REFRESH_PATH)).toHaveLength(1));
    releaseRefresh();
    const responses = await pending;

    expect(responses.map((r) => r.status)).toEqual([200, 200]);
    expect(api.callsTo(REFRESH_PATH)).toHaveLength(1);
  });

  it('no intenta refrescar ante un 401 de los endpoints de /auth', async () => {
    const { api, onEnd, authFetch } = setup({
      '/api/v1/auth/login': () => json(401, { error: { code: 'invalid_credentials', message: 'x' } }),
      [REFRESH_PATH]: () => refreshed('new'),
    });

    const response = await authFetch(new Request(`${BASE}/api/v1/auth/login`, { method: 'POST' }));

    expect(response.status).toBe(401);
    expect(api.callsTo(REFRESH_PATH)).toHaveLength(0);
    expect(onEnd).not.toHaveBeenCalled();
  });
});
