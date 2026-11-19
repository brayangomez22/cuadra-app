import { act, screen, waitFor } from '@testing-library/react';
import { aUser, apiError, fakeApi, json, signedIn, type Me } from '@/test/fakeApi';
import { aLocation, buckets, page } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';

const salesKey = 'GET /api/v1/reports/sales';
const sales = (total: string) => ({
  period: { from: '2026-11-19', to: '2026-11-19' },
  group_by: 'day' as const,
  items: [],
  totals: { sales_count: 4, net: '0.00', tax: '0.00', total },
});

// Every card of the dashboard answers; overrides replace one.
function homeApi(user: Me, overrides: Parameters<typeof fakeApi>[0] = {}) {
  return fakeApi({
    ...signedIn(user),
    [salesKey]: json(200, sales('400000.00')),
    'GET /api/v1/reports/team-control': json(200, {
      period: { from: '2026-09-28', to: '2026-11-19' },
      bucket: 'week' as const,
      users: [],
      series: [],
    }),
    'GET /api/v1/cash-sessions': json(200, page([])),
    'GET /api/v1/locations': json(200, { items: [aLocation()] }),
    'GET /api/v1/receivables/aging': json(200, { as_of: '2026-11-19', items: [], total: 0, totals: buckets({}) }),
    'GET /api/v1/inventory/reorder-alerts': json(200, page([])),
    ...overrides,
  });
}

const cards = ['Ventas de hoy', 'Cajas abiertas', 'Por reponer', 'Cartera vencida', 'Control del equipo esta semana'];

beforeEach(() => {
  vi.useFakeTimers({
    toFake: ['Date', 'setTimeout', 'clearTimeout', 'setInterval', 'clearInterval'],
    shouldAdvanceTime: true,
  });
  vi.setSystemTime(new Date('2026-11-19T15:30:00Z'));
});
afterEach(() => vi.useRealTimers());

describe('HomePage', () => {
  it.each(['owner', 'admin'] as const)('el %s ve el tablero con lo del día y lo que debe atender', async (role) => {
    homeApi(aUser(role));
    renderApp('/');

    for (const name of cards) expect(await screen.findByRole('region', { name })).toBeInTheDocument();
  });

  it('el cajero no ve el tablero: ve su saludo y cómo empezar a vender', async () => {
    const api = homeApi(aUser('cashier'));
    renderApp('/');

    expect(await screen.findByRole('heading', { name: /Hola/ })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Ir a vender' })).toHaveAttribute('href', '/vender');
    expect(screen.queryByRole('region', { name: 'Ventas de hoy' })).not.toBeInTheDocument();
    expect(api.calls(salesKey)).toHaveLength(0);
  });

  it('una tarjeta que falla muestra su error sin tumbar las demás', async () => {
    homeApi(aUser(), { [salesKey]: apiError(500, { code: 'internal', message: 'Falló' }) });
    renderApp('/');

    expect(await screen.findByText('No pudimos cargar las ventas de hoy')).toBeInTheDocument();
    expect(await screen.findByText('Nadie tiene cuentas vencidas.')).toBeInTheDocument();
    expect(await screen.findByText('Nada por reponer.')).toBeInTheDocument();
  });

  it('se actualiza solo cada 5 minutos', async () => {
    const api = homeApi(aUser());
    renderApp('/');

    expect(await screen.findByText('$ 400.000')).toBeInTheDocument();
    const before = api.calls(salesKey).length;
    await act(async () => {
      await vi.advanceTimersByTimeAsync(5 * 60 * 1000);
    });
    await waitFor(() => expect(api.calls(salesKey).length).toBeGreaterThan(before));
  });
});
