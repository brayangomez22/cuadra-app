import { renderHook } from '@testing-library/react';
import { errorShown } from '@/shared/observability';
import { startAnalytics, stopAnalytics, track } from '../tracker/tracker';
import { useAnalyticsSession } from './useAnalyticsSession';

vi.mock('../tracker/tracker');

const tenantId = '01920000-0000-7000-8000-0000000000aa';

describe('useAnalyticsSession', () => {
  afterEach(() => vi.clearAllMocks());

  it('empieza a enviar cuando /me trae el seudónimo', () => {
    renderHook(() => useAnalyticsSession({ analytics_id: 'abc123', tenant_id: tenantId }));

    expect(startAnalytics).toHaveBeenCalledWith({ analyticsId: 'abc123', tenantId });
  });

  it('no envía si la empresa no dio su consentimiento (sin seudónimo)', () => {
    renderHook(() => useAnalyticsSession({ tenant_id: tenantId }));

    expect(startAnalytics).not.toHaveBeenCalled();
    expect(stopAnalytics).toHaveBeenCalled();
  });

  it('deja de enviar al cerrar la sesión', () => {
    const initialProps: { user: Parameters<typeof useAnalyticsSession>[0] } = {
      user: { analytics_id: 'abc123', tenant_id: tenantId },
    };
    const { rerender } = renderHook(({ user }) => useAnalyticsSession(user), { initialProps });

    rerender({ user: null });

    expect(stopAnalytics).toHaveBeenCalled();
  });

  it('envía los errores que la app muestra, solo con su código', () => {
    renderHook(() => useAnalyticsSession({ analytics_id: 'abc123', tenant_id: tenantId }));

    errorShown('insufficient_stock');

    expect(track).toHaveBeenCalledWith('error_shown', { code: 'insufficient_stock', screen: '/' });
  });

  it('sin sesión de analítica no escucha los errores', () => {
    renderHook(() => useAnalyticsSession(null));

    errorShown('insufficient_stock');

    expect(track).not.toHaveBeenCalled();
  });
});
