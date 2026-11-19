import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { fakeApi, json, signedIn } from '@/test/fakeApi';
import { aLocation, aProduct, ids, page, plain, query } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { aReorderAlert } from '../../testing/reorderApi';

const alertsKey = 'GET /api/v1/inventory/reorder-alerts';

function reorderApi() {
  return fakeApi({
    ...signedIn(),
    'GET /api/v1/locations': json(200, { items: [aLocation(), aLocation({ id: ids.north, name: 'Sede norte' })] }),
    [alertsKey]: json(200, page([aReorderAlert({ location_id: ids.north })])),
    'GET /api/v1/products': json(200, page([aProduct()])),
  });
}

const lastQuery = (api: ReturnType<typeof reorderApi>) => query(api.calls(alertsKey).at(-1)!);

describe('ReorderPage con todas las sedes y su orden', () => {
  it('con todas las sedes pide las alertas sin sede y muestra la sede de cada fila', async () => {
    const api = reorderApi();
    renderApp('/inventario/reponer?sede=todas');

    const table = within(await screen.findByRole('table', { name: 'Por reponer en todas las sedes' }));
    const pipe = await table.findByRole('row', { name: /Tubo PVC/ });
    expect(plain(pipe.textContent)).toContain('Sede norte');
    expect(lastQuery(api).get('location_id')).toBeNull();
  });

  it('el selector de sede ofrece todas las sedes', async () => {
    const api = reorderApi();
    renderApp('/inventario/reponer');

    expect(await screen.findByRole('table', { name: 'Por reponer en Bodega principal' })).toBeInTheDocument();
    await userEvent.click(screen.getByRole('combobox', { name: 'Sede' }));
    await userEvent.click(await screen.findByRole('option', { name: 'Todas las sedes' }));
    expect(await screen.findByRole('table', { name: 'Por reponer en todas las sedes' })).toBeInTheDocument();
    await waitFor(() => expect(lastQuery(api).get('location_id')).toBeNull());
  });

  it('ordena por lo que más hay que pedir', async () => {
    const api = reorderApi();
    renderApp('/inventario/reponer');

    expect(await screen.findByRole('table', { name: 'Por reponer en Bodega principal' })).toBeInTheDocument();
    expect(lastQuery(api).get('sort')).toBeNull();
    await userEvent.click(screen.getByRole('combobox', { name: 'Ordenar por' }));
    await userEvent.click(await screen.findByRole('option', { name: 'Lo que más hay que pedir' }));
    await waitFor(() => expect(lastQuery(api).get('sort')).toBe('to_order'));
  });

  it('el orden se conserva en la URL', async () => {
    const api = reorderApi();
    renderApp('/inventario/reponer?sede=todas&orden=pedir');

    expect(await screen.findByRole('table', { name: 'Por reponer en todas las sedes' })).toBeInTheDocument();
    expect(lastQuery(api).get('sort')).toBe('to_order');
  });
});
