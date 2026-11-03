import { screen, within } from '@testing-library/react';
import { aUser, json, signedIn } from '@/test/fakeApi';
import { aLocation, ids, page, plain, query } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { cashApi } from '../../testing/cashApi';
import { aClosedSummary, anOpenSummary, otherCashierId, otherSessionId } from '../../testing/cashHistory';

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

  it('las cajas abiertas van arriba con quién, dónde y desde cuándo, y se cierran con un botón', async () => {
    historyApi();
    renderApp('/caja/cierres');

    const open = within(await screen.findByRole('region', { name: 'Cajas abiertas' }));
    const item = open.getByRole('listitem');
    expect(plain(item.textContent)).toContain('Pedro Ruiz');
    expect(plain(item.textContent)).toContain('Sede norte');
    expect(plain(item.textContent)).toContain('14/10/2026, 8:00 a. m.');
    expect(open.getByRole('link', { name: 'Cerrar la caja de Pedro Ruiz' })).toHaveAttribute(
      'href',
      `/caja/${otherSessionId}/cerrar`,
    );
    const table = within(screen.getByRole('table', { name: 'Cajas' }));
    expect(within(table.getByRole('row', { name: /Pedro Ruiz/ })).getAllByText('Abierta')).not.toHaveLength(0);
  });

  it('la caja abierta propia no se cierra desde aquí sino desde la caja', async () => {
    cashApi({
      ...signedIn({ ...aUser('admin'), id: otherCashierId }),
      'GET /api/v1/locations': json(200, { items: [aLocation(), aLocation({ id: ids.north, name: 'Sede norte' })] }),
      'GET /api/v1/cash-sessions': json(200, page([anOpenSummary()])),
    });
    renderApp('/caja/cierres');

    const open = within(await screen.findByRole('region', { name: 'Cajas abiertas' }));
    expect(open.getByRole('link', { name: 'Cerrar mi caja' })).toHaveAttribute('href', '/caja/cerrar');
  });

  it('sin cajas abiertas no muestra el bloque', async () => {
    cashApi({
      ...signedIn(aUser('admin')),
      'GET /api/v1/cash-sessions': json(200, page([aClosedSummary()])),
    });
    renderApp('/caja/cierres');

    await screen.findByRole('table', { name: 'Cajas' });
    expect(screen.queryByRole('region', { name: 'Cajas abiertas' })).not.toBeInTheDocument();
  });

  it('filtra por estado, sede y días desde la URL', async () => {
    const api = historyApi();
    renderApp(`/caja/cierres?estado=abiertas&sede=${ids.north}&desde=2026-10-01&hasta=2026-10-31`);

    await screen.findByRole('table', { name: 'Cajas' });
    const sent = query(api.calls('GET /api/v1/cash-sessions').find((r) => query(r).has('location_id'))!);
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
