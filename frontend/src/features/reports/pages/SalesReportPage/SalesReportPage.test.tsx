import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser } from '@/test/fakeApi';
import { renderApp } from '@/test/renderApp';
import { reportsApi } from '../../testing/reportsApi';
import { salesByMethod } from '../../testing/salesFixtures';
import { json } from '@/test/fakeApi';

const salesKey = 'GET /api/v1/reports/sales';
const url = '/reportes/ventas?desde=2026-11-01&hasta=2026-11-19';

describe('SalesReportPage', () => {
  it('muestra los totales del periodo y las ventas por día', async () => {
    const api = reportsApi();
    renderApp(url);

    const totals = await screen.findByRole('region', { name: 'Totales del periodo' });
    expect(within(totals).getByText('$ 892.500')).toBeInTheDocument();
    const row = within(screen.getByRole('cell', { name: /03\/11\/2026/ }).closest('tr')!);
    expect(row.getByText('$ 714.000')).toBeInTheDocument();
    const asked = new URL(api.calls(salesKey).at(-1)!.url);
    expect(asked.searchParams.get('from')).toBe('2026-11-01');
    expect(asked.searchParams.get('group_by')).toBe('day');
  });

  it('agrupa por medio de pago, con sus nombres y sin IVA aparte', async () => {
    const api = reportsApi({ [salesKey]: json(200, salesByMethod) });
    renderApp(url);

    await userEvent.click(await screen.findByRole('combobox', { name: 'Agrupar por' }));
    await userEvent.click(await screen.findByRole('option', { name: 'Medio de pago' }));

    expect(await screen.findByRole('cell', { name: /Nequi/ })).toBeInTheDocument();
    expect(screen.queryByRole('columnheader', { name: 'IVA' })).not.toBeInTheDocument();
    await waitFor(() => {
      expect(new URL(api.calls(salesKey).at(-1)!.url).searchParams.get('group_by')).toBe('payment_method');
    });
  });

  it('un periodo al revés avisa y no consulta', async () => {
    const api = reportsApi();
    renderApp('/reportes/ventas?desde=2026-11-19&hasta=2026-11-01');

    expect(await screen.findByText('El último día debe ser igual o posterior al primero.')).toBeInTheDocument();
    expect(api.calls(salesKey)).toHaveLength(0);
    expect(screen.getByRole('button', { name: 'Exportar a Excel' })).toBeDisabled();
  });

  it('exporta a Excel lo que está en pantalla', async () => {
    const api = reportsApi({ 'GET /api/v1/reports/export': new Response('xlsx', { status: 200 }) });
    URL.createObjectURL = vi.fn(() => 'blob:reporte');
    URL.revokeObjectURL = vi.fn();
    renderApp(url);

    await userEvent.click(await screen.findByRole('button', { name: 'Exportar a Excel' }));

    await waitFor(() => {
      const asked = new URL(api.calls('GET /api/v1/reports/export').at(-1)!.url);
      expect(asked.searchParams.get('report')).toBe('sales');
      expect(asked.searchParams.get('format')).toBe('xlsx');
      expect(asked.searchParams.get('to')).toBe('2026-11-19');
    });
  });

  it('el cajero no entra a los reportes', async () => {
    reportsApi({}, aUser('cashier'));
    renderApp(url);

    expect(await screen.findByText(/No tiene permiso/)).toBeInTheDocument();
  });
});
