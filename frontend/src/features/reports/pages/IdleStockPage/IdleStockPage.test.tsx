import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { json } from '@/test/fakeApi';
import { renderApp } from '@/test/renderApp';
import { idleReport, saleProfits } from '../../testing/productFixtures';
import { reportsApi } from '../../testing/reportsApi';

const idleKey = 'GET /api/v1/reports/idle-stock';

describe('IdleStockPage', () => {
  it('muestra lo que no se vende, desde cuándo y la plata quieta', async () => {
    reportsApi({ [idleKey]: json(200, idleReport) });
    renderApp('/reportes/sin-movimiento');

    const row = within((await screen.findByRole('cell', { name: /Tornillo 1\/4/ })).closest('tr')!);
    expect(row.getByText('200 und')).toBeInTheDocument();
    expect(row.getAllByText(/Nunca/).length).toBeGreaterThan(0);
    expect(within(screen.getByRole('region', { name: 'Plata quieta' })).getByText('$ 30.000')).toBeInTheDocument();
  });

  it('pide los días elegidos', async () => {
    const api = reportsApi({ [idleKey]: json(200, idleReport) });
    renderApp('/reportes/sin-movimiento');

    await userEvent.click(await screen.findByRole('combobox', { name: 'Sin vender hace' }));
    await userEvent.click(await screen.findByRole('option', { name: '90 días' }));

    await waitFor(() => {
      expect(new URL(api.calls(idleKey).at(-1)!.url).searchParams.get('days')).toBe('90');
    });
  });
});

describe('SaleProfitsPage', () => {
  it('muestra la utilidad del periodo y cada venta con su enlace', async () => {
    reportsApi({ 'GET /api/v1/reports/sale-profits': json(200, saleProfits) });
    renderApp('/reportes/utilidad?desde=2026-11-01&hasta=2026-11-19');

    const figures = await screen.findByRole('region', { name: 'Utilidad del periodo' });
    expect(within(figures).getByText('$ 5.000')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'N.º 128 · Mostrador centro' })).toHaveAttribute(
      'href',
      '/ventas/01920000-0000-7000-8000-000000000c01',
    );
  });
});
