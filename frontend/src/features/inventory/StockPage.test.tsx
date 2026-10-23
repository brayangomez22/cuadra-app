import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser, apiError, fakeApi, json, signedIn } from '@/test/fakeApi';
import {
  aCement,
  aGarment,
  aLocation,
  aProduct,
  aStockLevel,
  ids,
  page,
  plain,
  query,
  variantId,
} from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';

const locations = {
  items: [aLocation(), aLocation({ id: ids.north, name: 'Sede norte' })],
};

function stockApi(extra: Parameters<typeof fakeApi>[0] = {}) {
  return fakeApi({
    ...signedIn(),
    'GET /api/v1/locations': json(200, locations),
    [`GET /api/v1/locations/${ids.main}/stock`]: json(
      200,
      page([
        aStockLevel(),
        aStockLevel({ variant_id: ids.cement, product_id: ids.cement, quantity: '2.5', average_cost: '28000.00' }),
      ]),
    ),
    [`GET /api/v1/locations/${ids.north}/stock`]: json(
      200,
      page([aStockLevel({ location_id: ids.north, quantity: '7' })]),
    ),
    'GET /api/v1/products': json(200, page([aProduct(), aCement()])),
    ...extra,
  });
}

describe('StockPage', () => {
  it('muestra las existencias de la primera sede con el nombre del producto', async () => {
    const api = stockApi();
    renderApp('/inventario');

    const table = within(await screen.findByRole('table', { name: 'Existencias en Bodega principal' }));
    const cement = await table.findByRole('row', { name: /Cemento gris 50kg/ });

    expect(plain(cement.textContent)).toContain('2,5 bulto');
    expect(plain(cement.textContent)).toContain('$ 28.000');
    expect(table.getByRole('link', { name: 'Tubo PVC presión 1/2 pulgada' })).toHaveAttribute(
      'href',
      `/productos/${ids.pipe}/kardex?variante=${ids.pipe}&sede=${ids.main}`,
    );
    // The names of the whole page come in a single request, inactive products included.
    const [products] = api.calls('GET /api/v1/products');
    expect(query(products!).getAll('ids')).toEqual([ids.pipe, ids.cement]);
    expect(query(products!).get('status')).toBe('all');
    expect(api.calls('GET /api/v1/products')).toHaveLength(1);
  });

  it('muestra una fila por variante con su etiqueta y su SKU', async () => {
    const api = stockApi({
      [`GET /api/v1/locations/${ids.main}/stock`]: json(
        200,
        page([
          aStockLevel({ variant_id: variantId('M', 'Azul'), product_id: ids.garment, quantity: '4' }),
          aStockLevel({ variant_id: variantId('L', 'Rojo'), product_id: ids.garment, quantity: '9' }),
          aStockLevel(),
        ]),
      ),
      'GET /api/v1/products': json(200, page([aGarment(), aProduct()])),
    });
    renderApp('/inventario');

    const table = within(await screen.findByRole('table', { name: 'Existencias en Bodega principal' }));
    const medium = await table.findByRole('row', { name: /Camiseta básica \(Azul · M\)/ });

    expect(plain(medium.textContent)).toContain('CAM-M-AZUL');
    expect(plain(medium.textContent)).toContain('4 und');
    expect(plain(table.getByRole('row', { name: /Camiseta básica \(Rojo · L\)/ }).textContent)).toContain('9 und');
    expect(table.getByRole('link', { name: 'Camiseta básica (Azul · M)' })).toHaveAttribute(
      'href',
      `/productos/${ids.garment}/kardex?variante=${variantId('M', 'Azul')}&sede=${ids.main}`,
    );
    // Each product once, even with several variants on the page.
    expect(query(api.calls('GET /api/v1/products')[0]!).getAll('ids')).toEqual([ids.garment, ids.pipe]);
  });

  it('cambia de sede y pide sus existencias', async () => {
    const api = stockApi();
    const { router } = renderApp('/inventario');
    await screen.findByRole('table', { name: 'Existencias en Bodega principal' });

    await userEvent.click(screen.getByRole('combobox', { name: 'Sede' }));
    await userEvent.click(await screen.findByRole('option', { name: 'Sede norte' }));

    expect(await screen.findByRole('table', { name: 'Existencias en Sede norte' })).toBeInTheDocument();
    expect(api.calls(`GET /api/v1/locations/${ids.north}/stock`)).toHaveLength(1);
    expect(router.state.location.search).toBe(`?sede=${ids.north}`);
  });

  it('no muestra el costo promedio a quien no edita el catálogo', async () => {
    stockApi({ ...signedIn(aUser('warehouse')) });
    renderApp('/inventario');

    const table = within(await screen.findByRole('table', { name: 'Existencias en Bodega principal' }));

    expect(table.queryByRole('columnheader', { name: 'Costo promedio' })).not.toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Registrar entrada' })).toBeInTheDocument();
  });

  it('el cajero no ve Registrar entrada', async () => {
    stockApi({ ...signedIn(aUser('cashier')) });
    renderApp('/inventario');

    await screen.findByRole('table', { name: 'Existencias en Bodega principal' });

    expect(screen.queryByRole('link', { name: 'Registrar entrada' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Nueva sede' })).not.toBeInTheDocument();
  });

  it('avisa cuando la sede aún no tiene existencias', async () => {
    stockApi({ [`GET /api/v1/locations/${ids.main}/stock`]: json(200, page([])) });
    renderApp('/inventario');

    expect(
      await screen.findByRole('heading', { name: 'Bodega principal aún no tiene existencias' }),
    ).toBeInTheDocument();
  });

  it('sin sedes, el dueño crea la primera y ve sus existencias', async () => {
    let created = false;
    const api = stockApi({
      'GET /api/v1/locations': () => json(200, { items: created ? [aLocation()] : [] }),
      'POST /api/v1/locations': () => {
        created = true;
        return json(201, aLocation());
      },
    });
    renderApp('/inventario');

    await userEvent.click(await screen.findByRole('button', { name: 'Crear sede' }));
    const dialog = within(screen.getByRole('dialog', { name: 'Nueva sede' }));
    await userEvent.type(dialog.getByLabelText('Nombre de la sede'), 'Bodega principal');
    await userEvent.click(dialog.getByRole('button', { name: 'Crear sede' }));

    expect(await screen.findByRole('table', { name: 'Existencias en Bodega principal' })).toBeInTheDocument();
    expect(await api.calls('POST /api/v1/locations')[0]?.json()).toEqual({ name: 'Bodega principal' });
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
  });

  it('muestra en el campo un nombre de sede repetido', async () => {
    stockApi({
      'POST /api/v1/locations': apiError(409, {
        code: 'location_name_taken',
        message: 'Ya existe una sede con ese nombre',
      }),
    });
    renderApp('/inventario');

    await userEvent.click(await screen.findByRole('button', { name: 'Nueva sede' }));
    const dialog = within(screen.getByRole('dialog', { name: 'Nueva sede' }));
    await userEvent.type(dialog.getByLabelText('Nombre de la sede'), 'bodega principal');
    await userEvent.click(dialog.getByRole('button', { name: 'Crear sede' }));

    await waitFor(() =>
      expect(dialog.getByLabelText('Nombre de la sede')).toHaveAccessibleDescription(
        expect.stringContaining('Ya existe una sede con ese nombre'),
      ),
    );
  });

  it('sin sedes, el bodeguero ve que debe pedirla al dueño', async () => {
    stockApi({ ...signedIn(aUser('warehouse')), 'GET /api/v1/locations': json(200, { items: [] }) });
    renderApp('/inventario');

    expect(await screen.findByRole('heading', { name: 'Aún no hay sedes' })).toBeInTheDocument();
    expect(screen.getByText(/Pida al dueño/)).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Crear sede' })).not.toBeInTheDocument();
  });
});
