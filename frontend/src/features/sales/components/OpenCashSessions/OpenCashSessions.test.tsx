import { screen, within } from '@testing-library/react';
import { aUser, fakeApi, json, signedIn } from '@/test/fakeApi';
import { aLocation, ids, page, query } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { anOpenSummary, otherSessionId } from '../../testing/cashHistory';

const sessionsKey = 'GET /api/v1/cash-sessions';

function cashApi(sessions = [anOpenSummary()], user = aUser()) {
  return fakeApi({
    ...signedIn(user),
    [sessionsKey]: json(200, page(sessions)),
    'GET /api/v1/locations': json(200, { items: [aLocation(), aLocation({ id: ids.north, name: 'Sede norte' })] }),
  });
}

const card = async () => within(await screen.findByRole('region', { name: 'Cajas abiertas' }));

describe('OpenCashSessions en el inicio', () => {
  it('muestra las cajas abiertas con su cajero, su sede y el botón para cerrarlas', async () => {
    const api = cashApi();
    renderApp('/');

    const drawers = await card();
    expect(await drawers.findByText('Pedro Ruiz')).toBeInTheDocument();
    expect(await drawers.findByText(/Sede norte/)).toBeInTheDocument();
    expect(drawers.getByRole('link', { name: 'Cerrar la caja de Pedro Ruiz' })).toHaveAttribute(
      'href',
      `/caja/${otherSessionId}/cerrar`,
    );
    expect(query(api.calls(sessionsKey)[0]!).get('status')).toBe('open');
  });

  it('sin cajas abiertas lo dice', async () => {
    cashApi([]);
    renderApp('/');

    expect(await screen.findByText('No hay cajas abiertas.')).toBeInTheDocument();
  });

  it('quien no administra la caja no la ve ni la pide', async () => {
    const api = cashApi([anOpenSummary()], aUser('cashier'));
    renderApp('/');

    expect(await screen.findByRole('heading', { name: /Hola/ })).toBeInTheDocument();
    expect(screen.queryByRole('region', { name: 'Cajas abiertas' })).not.toBeInTheDocument();
    expect(api.calls(sessionsKey)).toHaveLength(0);
  });
});
