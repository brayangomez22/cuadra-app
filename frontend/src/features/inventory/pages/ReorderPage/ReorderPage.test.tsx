import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser, fakeApi, json, signedIn } from '@/test/fakeApi';
import { aGarment, aLocation, aProduct, ids, page, plain, query, variantId } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { aReorderAlert } from '../../testing/reorderApi';

const alertsKey = 'GET /api/v1/inventory/reorder-alerts';

function reorderApi(alerts = page([aReorderAlert()]), user = aUser()) {
  return fakeApi({
    ...signedIn(user),
    'GET /api/v1/locations': json(200, { items: [aLocation(), aLocation({ id: ids.north, name: 'Sede norte' })] }),
    [alertsKey]: json(200, alerts),
    'GET /api/v1/products': json(200, page([aProduct(), aGarment()])),
  });
}

describe('ReorderPage', () => {
  it('lista lo que hay que reponer en la sede con la cantidad sugerida', async () => {
    const api = reorderApi(
      page([
        aReorderAlert({
          variant_id: variantId('M', 'Azul'),
          product_id: ids.garment,
          quantity: '1',
          min_quantity: '2',
          max_quantity: '6',
          suggested_quantity: '5',
        }),
        aReorderAlert(),
      ]),
      aUser('cashier'),
    );
    renderApp('/inventario/reponer');

    const table = within(await screen.findByRole('table', { name: 'Por reponer en Bodega principal' }));
    const shirt = await table.findByRole('row', { name: /Camiseta básica \(Azul · M\)/ });
    expect(plain(shirt.textContent)).toContain('1 und');
    expect(plain(shirt.textContent)).toContain('5 und');
    const pipe = table.getByRole('row', { name: /Tubo PVC/ });
    expect(plain(pipe.textContent)).toMatch(/3 und.*10 und.*50 und.*47 und/);
    expect(table.getByRole('link', { name: 'Tubo PVC presión 1/2 pulgada' })).toHaveAttribute(
      'href',
      `/productos/${ids.pipe}/kardex?variante=${ids.pipe}&sede=${ids.main}`,
    );
    expect(query(api.calls(alertsKey)[0]!).get('location_id')).toBe(ids.main);
  });

  it('cambia de sede desde la URL y el selector', async () => {
    const api = reorderApi();
    renderApp(`/inventario/reponer?sede=${ids.north}`);

    expect(await screen.findByRole('table', { name: 'Por reponer en Sede norte' })).toBeInTheDocument();
    expect(query(api.calls(alertsKey)[0]!).get('location_id')).toBe(ids.north);

    await userEvent.click(screen.getByRole('combobox', { name: 'Sede' }));
    await userEvent.click(await screen.findByRole('option', { name: 'Bodega principal' }));
    await waitFor(() => expect(query(api.calls(alertsKey).at(-1)!).get('location_id')).toBe(ids.main));
  });

  it('sin alertas lo dice y lleva al inventario', async () => {
    reorderApi(page([]));
    renderApp('/inventario/reponer');

    expect(await screen.findByRole('heading', { name: 'Nada por reponer en Bodega principal' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Ver inventario' })).toHaveAttribute(
      'href',
      `/inventario?sede=${ids.main}`,
    );
  });
});
