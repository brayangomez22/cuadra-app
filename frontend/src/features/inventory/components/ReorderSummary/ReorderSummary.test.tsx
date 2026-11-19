import { screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser, apiError, fakeApi, json, signedIn } from '@/test/fakeApi';
import { aLocation, aProduct, page, query } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { aReorderAlert } from '../../testing/reorderApi';

const alertsKey = 'GET /api/v1/inventory/reorder-alerts';

type Routes = Parameters<typeof fakeApi>[0];

function reorderApi(routes: Routes = {}, user = aUser()) {
  return fakeApi({
    ...signedIn(user),
    [alertsKey]: json(200, page([aReorderAlert()], 7)),
    'GET /api/v1/locations': json(200, { items: [aLocation()] }),
    'GET /api/v1/products': json(200, page([aProduct()])),
    ...routes,
  });
}

const card = async () => within(await screen.findByRole('region', { name: 'Por reponer' }));

describe('ReorderSummary en el inicio', () => {
  it('dice cuántos productos hay por reponer sumando todas las sedes', async () => {
    const api = reorderApi();
    renderApp('/');

    const reorder = await card();
    expect(await reorder.findByText('7 productos')).toBeInTheDocument();
    expect(reorder.getByText(/sumando todas las sedes/)).toBeInTheDocument();
    const [request] = api.calls(alertsKey);
    expect(query(request!).get('location_id')).toBeNull();
    expect(query(request!).get('limit')).toBe('1');
  });

  it('enlaza a la lista de todas las sedes, que da el mismo total', async () => {
    reorderApi();
    renderApp('/');

    const link = await (await card()).findByRole('link', { name: 'Ver qué pedir' });
    expect(link).toHaveAttribute('href', '/inventario/reponer?sede=todas');
    await userEvent.click(link);
    expect(await screen.findByRole('table', { name: 'Por reponer en todas las sedes' })).toBeInTheDocument();
    expect(await screen.findByText(/de 7$/)).toBeInTheDocument();
  });

  it('con uno solo habla en singular', async () => {
    reorderApi({ [alertsKey]: json(200, page([aReorderAlert()])) });
    renderApp('/');

    expect(await (await card()).findByText('1 producto')).toBeInTheDocument();
  });

  it('el bodeguero también lo ve en su inicio', async () => {
    reorderApi({ [alertsKey]: json(200, page([aReorderAlert()], 3)) }, aUser('warehouse'));
    renderApp('/');

    expect(await (await card()).findByText('3 productos')).toBeInTheDocument();
  });

  it('sin alertas dice que no hay nada por reponer', async () => {
    reorderApi({ [alertsKey]: json(200, page([])) });
    renderApp('/');

    expect(await (await card()).findByText('Nada por reponer.')).toBeInTheDocument();
  });

  it('si falla muestra el error dentro de la tarjeta', async () => {
    reorderApi({ [alertsKey]: apiError(500, { code: 'internal', message: 'Falló' }) });
    renderApp('/');

    expect(await (await card()).findByText('No pudimos revisar lo que hay por reponer')).toBeInTheDocument();
  });
});
