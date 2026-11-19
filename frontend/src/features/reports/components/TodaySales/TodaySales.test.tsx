import { screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser, fakeApi, json, signedIn } from '@/test/fakeApi';
import { plain, query } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { dashboardNow, salesOfDay } from '../../testing/dashboardFixtures';

const salesKey = 'GET /api/v1/reports/sales';

// Today the business sold 1.200.000 in 10 sales; last Thursday, at this time, 1.000.000.
function salesApi({
  today = salesOfDay('2026-11-19', '1200000.00', 10),
  lastWeek = salesOfDay('2026-11-12', '1000000.00', 8),
} = {}) {
  return fakeApi({
    ...signedIn(aUser('admin')),
    [salesKey]: (request) => json(200, query(request).get('from') === '2026-11-19' ? today : lastWeek),
  });
}

const card = async () => within(await screen.findByRole('region', { name: 'Ventas de hoy' }));

beforeEach(() => {
  vi.useFakeTimers({ toFake: ['Date'] });
  vi.setSystemTime(dashboardNow);
});
afterEach(() => vi.useRealTimers());

describe('TodaySales en el inicio', () => {
  it('compara las ventas de hoy con las del mismo día de la semana pasada hasta la misma hora', async () => {
    const api = salesApi();
    renderApp('/');

    const sales = await card();
    expect(await sales.findByText('$ 1.200.000')).toBeInTheDocument();
    expect(plain(sales.getByText(/el jueves pasado/).textContent)).toContain(
      '20 % más que el jueves pasado a esta hora ($ 1.000.000)',
    );
    const [today, lastWeek] = api.calls(salesKey).map((request) => query(request));
    expect([today?.get('from'), today?.get('to'), today?.get('until')]).toEqual(['2026-11-19', '2026-11-19', '10:30']);
    expect([lastWeek?.get('from'), lastWeek?.get('to'), lastWeek?.get('until')]).toEqual([
      '2026-11-12',
      '2026-11-12',
      '10:30',
    ]);
  });

  it('muestra el número de ventas y el ticket promedio', async () => {
    salesApi();
    renderApp('/');

    const sales = await card();
    expect(plain((await sales.findByText('Ventas', { selector: 'dt' })).nextElementSibling?.textContent)).toBe('10');
    expect(plain(sales.getByText('Ticket promedio', { selector: 'dt' }).nextElementSibling?.textContent)).toBe(
      '$ 120.000',
    );
  });

  it('sin ventas hoy dice cuánto llevaba la semana pasada, sin porcentaje ni ticket promedio', async () => {
    salesApi({ today: salesOfDay('2026-11-19', '0.00', 0) });
    renderApp('/');

    const sales = await card();
    expect(
      await sales.findByText(/Aún no hay ventas hoy. El jueves pasado a esta hora llevaba \$ 1.000.000/),
    ).toBeInTheDocument();
    expect(sales.queryByText('Ticket promedio')).not.toBeInTheDocument();
  });

  it('si vende menos lo dice con el porcentaje', async () => {
    salesApi({ today: salesOfDay('2026-11-19', '800000.00', 6) });
    renderApp('/');

    expect(await (await card()).findByText(/20 % menos que el jueves pasado a esta hora/)).toBeInTheDocument();
  });

  it('si la semana pasada no había vendido a esta hora, no inventa un porcentaje', async () => {
    salesApi({ lastWeek: salesOfDay('2026-11-12', '0.00', 0) });
    renderApp('/');

    expect(await (await card()).findByText('El jueves pasado a esta hora no había ventas.')).toBeInTheDocument();
  });

  it('enlaza al reporte de ventas del día, que da el mismo total', async () => {
    salesApi();
    renderApp('/');

    const link = await (await card()).findByRole('link', { name: 'Ver las ventas de hoy' });
    expect(link).toHaveAttribute('href', '/reportes/ventas?desde=2026-11-19&hasta=2026-11-19');
    await userEvent.click(link);
    const figures = within(await screen.findByRole('region', { name: 'Totales del periodo' }));
    expect(plain((await figures.findByText('Total', { selector: 'dt' })).nextElementSibling?.textContent)).toBe(
      '$ 1.200.000',
    );
  });
});
