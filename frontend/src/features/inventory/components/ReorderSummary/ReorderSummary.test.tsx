import { screen, waitFor } from '@testing-library/react';
import { aUser, apiError, fakeApi, json, signedIn } from '@/test/fakeApi';
import { page, query } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { aReorderAlert } from '../../testing/reorderApi';

const alertsKey = 'GET /api/v1/inventory/reorder-alerts';

describe('ReorderSummary en el inicio', () => {
  it('avisa cuántos productos hay por reponer en todas las sedes', async () => {
    const api = fakeApi({ ...signedIn(), [alertsKey]: json(200, page([aReorderAlert()], 7)) });
    renderApp('/');

    expect(await screen.findByText('7 productos por reponer')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Ver qué pedir' })).toHaveAttribute('href', '/inventario/reponer');
    const [request] = api.calls(alertsKey);
    expect(query(request!).get('location_id')).toBeNull();
    expect(query(request!).get('limit')).toBe('1');
  });

  it('con uno solo habla en singular', async () => {
    fakeApi({ ...signedIn(), [alertsKey]: json(200, page([aReorderAlert()])) });
    renderApp('/');

    expect(await screen.findByText('1 producto por reponer')).toBeInTheDocument();
  });

  it.each([
    ['sin alertas', json(200, page([]))],
    ['si la consulta falla', apiError(500, { code: 'internal', message: 'Falló' })],
  ])('%s no muestra nada', async (_, reply) => {
    const api = fakeApi({ ...signedIn(aUser()), [alertsKey]: reply });
    renderApp('/');

    expect(await screen.findByRole('heading', { name: /Hola/ })).toBeInTheDocument();
    await waitFor(() => expect(api.calls(alertsKey)).toHaveLength(1));
    expect(screen.queryByText(/por reponer/)).not.toBeInTheDocument();
  });
});
