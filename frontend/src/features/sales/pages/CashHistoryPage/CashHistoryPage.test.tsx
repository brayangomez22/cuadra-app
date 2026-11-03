import { screen, within } from '@testing-library/react';
import { aUser, json, signedIn } from '@/test/fakeApi';
import { aLocation, ids, page, plain, query } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { cashApi } from '../../testing/cashApi';
import { aClosedSummary, anOpenSummary, otherSessionId } from '../../testing/cashHistory';

function historyApi() {
  return cashApi({
    ...signedIn(aUser('admin')),
    'GET /api/v1/locations': json(200, { items: [aLocation(), aLocation({ id: ids.north, name: 'Sede norte' })] }),
    'GET /api/v1/cash-sessions': json(200, page([anOpenSummary(), aClosedSummary()])),
  });
}

describe('CashHistoryPage', () => {
  it('muestra cada caja con su cajero, su sede, lo vendido y cómo cuadró', async () => {
    historyApi();
    renderApp('/caja/cierres');

    const table = within(await screen.findByRole('table', { name: 'Cajas' }));
    const closed = await table.findByRole('row', { name: /Laura Pérez/ });
    expect(plain(closed.textContent)).toContain('Bodega principal');
    expect(plain(closed.textContent)).toContain('$ 105.000');
    expect(within(closed).getAllByText('Faltante')).not.toHaveLength(0);
    expect(plain(closed.textContent)).toContain('-$ 3.000');
    expect(within(closed).getByRole('link', { name: /13\/10\/2026/ })).toHaveAttribute(
      'href',
      `/caja/${ids.cashSession}`,
    );
  });

  it('una caja que otro dejó abierta se puede cerrar desde aquí', async () => {
    historyApi();
    renderApp('/caja/cierres');

    const table = within(await screen.findByRole('table', { name: 'Cajas' }));
    const open = await table.findByRole('row', { name: /Pedro Ruiz/ });
    expect(within(open).getAllByText('Abierta')).not.toHaveLength(0);
    expect(within(open).getByRole('link', { name: 'Cerrar la caja de Pedro Ruiz' })).toHaveAttribute(
      'href',
      `/caja/${otherSessionId}/cerrar`,
    );
  });

  it('filtra por estado, sede y días desde la URL', async () => {
    const api = historyApi();
    renderApp(`/caja/cierres?estado=abiertas&sede=${ids.north}&desde=2026-10-01&hasta=2026-10-31`);

    await screen.findByRole('table', { name: 'Cajas' });
    const sent = query(api.calls('GET /api/v1/cash-sessions')[0]!);
    expect(sent.get('status')).toBe('open');
    expect(sent.get('location_id')).toBe(ids.north);
    expect(sent.get('from')).toBe('2026-10-01');
    expect(sent.get('to')).toBe('2026-10-31');
  });

  it('el cajero no ve el historial', async () => {
    cashApi();
    renderApp('/caja/cierres');

    expect(await screen.findByRole('heading', { name: 'No tiene permiso para ver esta sección' })).toBeInTheDocument();
  });
});
