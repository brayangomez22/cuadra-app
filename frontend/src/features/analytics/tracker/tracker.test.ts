import { startAnalytics, stopAnalytics, track, trackOnStart } from './tracker';

// The PostHog client the tracker loads, replaced by mocks.
const posthog = vi.hoisted(() => ({
  init: vi.fn(),
  identify: vi.fn(),
  group: vi.fn(),
  capture: vi.fn(),
  reset: vi.fn(),
}));

vi.mock('posthog-js', () => ({ default: posthog }));

const session = { analyticsId: '3f2a9c0d1e4b5a6978c0d1e2f3a4b5c6', tenantId: '01920000-0000-7000-8000-0000000000aa' };

afterEach(async () => {
  await stopAnalytics();
  vi.unstubAllEnvs();
  vi.clearAllMocks();
});

describe('tracker', () => {
  it('sin la llave de PostHog no carga nada ni envía eventos', async () => {
    vi.stubEnv('VITE_POSTHOG_KEY', '');

    await startAnalytics(session);
    await track('sale_confirmed', { lines: 2 });

    expect(posthog.init).not.toHaveBeenCalled();
    expect(posthog.capture).not.toHaveBeenCalled();
  });

  it('sin sesión de analítica no envía eventos', async () => {
    vi.stubEnv('VITE_POSTHOG_KEY', 'phc_test');

    await track('sale_confirmed', { lines: 2 });

    expect(posthog.capture).not.toHaveBeenCalled();
  });

  it('inicia PostHog sin autocaptura, grabación ni cookies, con el seudónimo y la empresa', async () => {
    vi.stubEnv('VITE_POSTHOG_KEY', 'phc_test');

    await startAnalytics(session);

    expect(posthog.init).toHaveBeenCalledWith(
      'phc_test',
      expect.objectContaining({
        autocapture: false,
        capture_pageview: false,
        disable_session_recording: true,
        persistence: 'memory',
        mask_all_text: true,
      }),
    );
    expect(posthog.identify).toHaveBeenCalledWith(session.analyticsId);
    expect(posthog.group).toHaveBeenCalledWith('tenant', session.tenantId);
  });

  it('envía solo las propiedades permitidas del evento', async () => {
    vi.stubEnv('VITE_POSTHOG_KEY', 'phc_test');
    await startAnalytics(session);

    // Un almacén de calzado confirma una venta; la pantalla pasa de más por error.
    await track('sale_confirmed', { lines: 3, with_customer: true, customer_name: 'Ana Gómez' } as never);

    expect(posthog.capture).toHaveBeenCalledWith('sale_confirmed', {
      catalog_version: 2,
      lines: 3,
      with_customer: true,
    });
  });

  it('al cerrar la sesión deja de enviar y olvida al usuario', async () => {
    vi.stubEnv('VITE_POSTHOG_KEY', 'phc_test');
    await startAnalytics(session);

    await stopAnalytics();
    await track('feedback_sent', { screen: '/vender' });

    expect(posthog.reset).toHaveBeenCalled();
    expect(posthog.capture).not.toHaveBeenCalled();
  });

  it('un evento pendiente se envía cuando arranca la sesión de analítica (el registro con consentimiento)', async () => {
    vi.stubEnv('VITE_POSTHOG_KEY', 'phc_test');

    await trackOnStart('signup_completed', { business_template: 'ropa_calzado' });
    expect(posthog.capture).not.toHaveBeenCalled();
    await startAnalytics(session);

    expect(posthog.capture).toHaveBeenCalledWith('signup_completed', {
      catalog_version: 2,
      business_template: 'ropa_calzado',
    });
  });

  it('un evento pendiente se descarta si la sesión no arranca (sin consentimiento)', async () => {
    vi.stubEnv('VITE_POSTHOG_KEY', 'phc_test');

    await trackOnStart('signup_completed', { business_template: 'ferreteria' });
    await stopAnalytics();
    await startAnalytics(session);

    expect(posthog.capture).not.toHaveBeenCalled();
  });

  it('con la sesión ya iniciada, un evento pendiente se envía enseguida', async () => {
    vi.stubEnv('VITE_POSTHOG_KEY', 'phc_test');
    await startAnalytics(session);

    await trackOnStart('signup_completed', { business_template: 'ferreteria' });

    expect(posthog.capture).toHaveBeenCalledTimes(1);
  });
});
