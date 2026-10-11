import * as Sentry from '@sentry/react';
import { recordFailedRequest, setReportingUser } from './sentry';

vi.mock('@sentry/react', () => ({ setTag: vi.fn(), setUser: vi.fn() }));

afterEach(() => {
  vi.clearAllMocks();
});

describe('recordFailedRequest', () => {
  it('guarda el X-Request-ID de una respuesta 5xx como tag', () => {
    recordFailedRequest(new Response(null, { status: 503, headers: { 'X-Request-ID': '4bf92f3577b34da6' } }));

    expect(Sentry.setTag).toHaveBeenCalledWith('last_failed_request_id', '4bf92f3577b34da6');
  });

  it('ignora las respuestas exitosas y los errores 4xx', () => {
    recordFailedRequest(new Response(null, { status: 200, headers: { 'X-Request-ID': 'a' } }));
    recordFailedRequest(new Response(null, { status: 409, headers: { 'X-Request-ID': 'b' } }));

    expect(Sentry.setTag).not.toHaveBeenCalled();
  });
});

describe('setReportingUser', () => {
  it('con sesión marca tenant_id y user.id, sin email ni nombre', () => {
    setReportingUser({ id: 'user-1', tenant_id: 'tenant-1' });

    expect(Sentry.setTag).toHaveBeenCalledWith('tenant_id', 'tenant-1');
    expect(Sentry.setUser).toHaveBeenCalledWith({ id: 'user-1' });
  });

  it('al terminar la sesión los limpia', () => {
    setReportingUser(null);

    expect(Sentry.setTag).toHaveBeenCalledWith('tenant_id', undefined);
    expect(Sentry.setUser).toHaveBeenCalledWith(null);
  });
});
