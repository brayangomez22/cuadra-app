import type { Breadcrumb, ErrorEvent } from '@sentry/react';
import { scrubBreadcrumb, scrubEvent } from './scrub';

const jwt =
  'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIwMTkyIn0.Sfl-KxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c';
const refreshToken = '0192f0a1-1234-7abc-8def-0123456789ab.q3J0Zm9vYmFyYmF6cXV4cXV1eGNvcmdlZ3JhdWx0YWJj';

// expectGone fails if any of the values survived anywhere in the scrubbed object.
function expectGone(value: unknown, gone: string[]) {
  const raw = JSON.stringify(value);
  for (const s of gone) expect(raw).not.toContain(s);
}

describe('scrubEvent', () => {
  it('elimina emails del mensaje y de las excepciones', () => {
    const event = scrubEvent({
      type: undefined,
      message: 'usuario ana.perez@ferreteria.co no encontrado',
      exception: { values: [{ type: 'Error', value: 'No existe ana+pos@gmail.com' }] },
    });

    expectGone(event, ['ana.perez@ferreteria.co', 'ana+pos@gmail.com']);
    expect(event.message).toBe('usuario [email] no encontrado');
    expect(event.exception?.values?.[0]?.value).toBe('No existe [email]');
  });

  it('elimina tokens JWT, Bearer y refresh tokens', () => {
    const event = scrubEvent({
      type: undefined,
      message: `token rechazado: ${jwt}`,
      exception: { values: [{ value: `Authorization: Bearer abc123.def-456_ghi y ${refreshToken}` }] },
    });

    expectGone(event, [jwt, 'abc123.def-456_ghi', refreshToken]);
    expect(event.message).toBe('token rechazado: [token]');
    expect(event.exception?.values?.[0]?.value).toContain('Bearer [token]');
  });

  it('elimina contraseñas escritas en el texto', () => {
    const event = scrubEvent({ type: undefined, message: 'login failed password=hunter2 secret: s3cr3t-value' });

    expectGone(event, ['hunter2', 's3cr3t-value']);
  });

  it('reemplaza valores de claves sensibles en extra y contexts', () => {
    const event = scrubEvent({
      type: undefined,
      extra: {
        password: 'hunter2',
        nested: { refresh_token: 'opaque-value', items: 3 },
        customerEmail: 'cliente@correo.com',
        count: 3,
      },
      contexts: { app: { api_key: 'k-123456', name: 'cuadra-web' } },
    });

    expectGone(event, ['hunter2', 'opaque-value', 'cliente@correo.com', 'k-123456']);
    expect(event.extra).toMatchObject({ password: '[filtered]', count: 3, nested: { items: 3 } });
    expect(event.contexts?.app?.name).toBe('cuadra-web');
  });

  it('elimina body, cookies, headers sensibles y query string de la request', () => {
    const event = scrubEvent({
      type: undefined,
      request: {
        url: 'https://app.cuadra.co/ingresar?email=ana@correo.com',
        method: 'POST',
        data: { email: 'ana@correo.com', password: 'hunter2' },
        query_string: 'email=ana@correo.com',
        cookies: { cuadra_refresh: refreshToken },
        headers: { Authorization: `Bearer ${jwt}`, 'User-Agent': 'Mozilla/5.0', Referer: 'https://app.cuadra.co/?t=1' },
      },
    });

    expectGone(event, ['ana@correo.com', 'hunter2', refreshToken, jwt, '?t=1']);
    expect(event.request).toEqual({
      url: 'https://app.cuadra.co/ingresar',
      method: 'POST',
      headers: { 'User-Agent': 'Mozilla/5.0' },
    });
  });

  it('deja user.id pero borra email, username e IP', () => {
    const event = scrubEvent({
      type: undefined,
      user: { id: '0192f0a1-1234-7abc-8def-0123456789ab', email: 'ana@correo.com', username: 'ana', ip_address: '181.50.10.20' },
    });

    expect(event.user).toEqual({ id: '0192f0a1-1234-7abc-8def-0123456789ab' });
  });

  it('filtra los breadcrumbs que vienen dentro del evento', () => {
    const event = scrubEvent({
      type: undefined,
      breadcrumbs: [{ message: 'login de ana@correo.com', data: { url: `/api/v1/me?access=${jwt}`, status_code: 401 } }],
    });

    expectGone(event, ['ana@correo.com', jwt]);
    expect(event.breadcrumbs?.[0]?.data).toEqual({ url: '/api/v1/me', status_code: 401 });
  });

  it('no modifica un evento sin datos sensibles', () => {
    const original: ErrorEvent = {
      type: undefined,
      message: 'stock insuficiente del producto 0192f0a1-1234-7abc-8def-0123456789ab',
      exception: { values: [{ type: 'TypeError', value: "Cannot read properties of undefined (reading 'price')" }] },
      tags: { tenant_id: '0192f0a1-0000-7000-8000-0000000000aa', last_failed_request_id: '4bf92f3577b34da6' },
      user: { id: '0192f0a1-0000-7000-8000-0000000000bb' },
    };

    expect(scrubEvent(structuredClone(original))).toEqual(original);
  });
});

describe('scrubBreadcrumb', () => {
  it('quita la query string de las URLs de fetch y navegación', () => {
    const crumb: Breadcrumb = {
      category: 'fetch',
      data: { method: 'GET', url: '/api/v1/products?search=ana@correo.com', status_code: 200 },
    };

    expect(scrubBreadcrumb(crumb)?.data).toEqual({ method: 'GET', url: '/api/v1/products', status_code: 200 });
  });

  it('filtra emails y tokens del mensaje de la consola', () => {
    const crumb = scrubBreadcrumb({ category: 'console', message: `token ${jwt} de ana@correo.com` });

    expect(crumb?.message).toBe('token [token] de [email]');
  });
});
