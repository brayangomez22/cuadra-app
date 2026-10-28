import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser, fakeApi, json, signedIn } from '@/test/fakeApi';
import { aCement, aGarment, aProduct, page, plain, query, withoutCost } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';

// Types without pauses between keys, like a fast typist: the debounce sees one word.
const typeFast = (element: HTMLElement, text: string) => userEvent.setup({ delay: null }).type(element, text);

async function productsTable() {
  return within(await screen.findByRole('table', { name: 'Productos' }));
}

describe('ProductListPage', () => {
  it('muestra los productos con el precio con IVA en pesos', async () => {
    fakeApi({ ...signedIn(), 'GET /api/v1/products': json(200, page([aProduct(), aCement()])) });
    renderApp('/productos');

    const table = await productsTable();
    const row = table.getByRole('row', { name: /Tubo PVC presión/ });

    expect(within(row).getByText('TUB-PVC-12')).toBeInTheDocument();
    expect(plain(row.textContent)).toContain('$ 11.900');
    expect(table.getByRole('row', { name: /Cemento gris 50kg/ })).toBeInTheDocument();
  });

  it('la búsqueda llama a la API una sola vez tras el debounce', async () => {
    const api = fakeApi({
      ...signedIn(),
      'GET /api/v1/products': (request) =>
        json(200, query(request).get('q') === 'tubo' ? page([aProduct()]) : page([aProduct(), aCement()])),
    });
    renderApp('/productos');
    await productsTable();

    await typeFast(screen.getByRole('searchbox', { name: 'Buscar' }), 'tubo');

    await waitFor(() => expect(screen.queryByRole('row', { name: /Cemento/ })).not.toBeInTheDocument());
    const searched = api
      .calls('GET /api/v1/products')
      .map((r) => query(r).get('q'))
      .filter(Boolean);
    expect(searched).toEqual(['tubo']);
  });

  it('vuelve a la primera página al buscar y conserva la búsqueda en la URL', async () => {
    const api = fakeApi({ ...signedIn(), 'GET /api/v1/products': json(200, page([aProduct()], 45)) });
    const { router } = renderApp('/productos');
    await userEvent.click(await screen.findByRole('button', { name: 'Siguiente' }));
    await waitFor(() => expect(query(api.calls('GET /api/v1/products').at(-1)!).get('offset')).toBe('20'));

    await typeFast(screen.getByRole('searchbox', { name: 'Buscar' }), 'cemento');

    await waitFor(() => expect(query(api.calls('GET /api/v1/products').at(-1)!).get('q')).toBe('cemento'));
    expect(query(api.calls('GET /api/v1/products').at(-1)!).get('offset') ?? '0').toBe('0');
    expect(router.state.location.search).toBe('?q=cemento');
  });

  it('pagina con siguiente y anterior', async () => {
    const api = fakeApi({ ...signedIn(), 'GET /api/v1/products': json(200, page([aProduct()], 45)) });
    renderApp('/productos');
    const pages = within(await screen.findByRole('navigation', { name: 'Páginas de productos' }));
    expect(pages.getByText('1–20 de 45')).toBeInTheDocument();

    await userEvent.click(pages.getByRole('button', { name: 'Siguiente' }));
    expect(await pages.findByText('21–40 de 45')).toBeInTheDocument();
    await userEvent.click(pages.getByRole('button', { name: 'Anterior' }));
    expect(await pages.findByText('1–20 de 45')).toBeInTheDocument();

    const offsets = api.calls('GET /api/v1/products').map((r) => query(r).get('offset') ?? '0');
    expect(offsets).toEqual(['0', '20', '0']);
  });

  it('pide los inactivos al cambiar el filtro de estado', async () => {
    const api = fakeApi({ ...signedIn(), 'GET /api/v1/products': json(200, page([aProduct({ active: false })])) });
    renderApp('/productos');
    await productsTable();

    await userEvent.click(screen.getByRole('combobox', { name: 'Estado' }));
    await userEvent.click(await screen.findByRole('option', { name: 'Inactivos' }));

    await waitFor(() => expect(query(api.calls('GET /api/v1/products').at(-1)!).get('status')).toBe('inactive'));
    expect(await screen.findByText('Inactivo')).toBeInTheDocument();
  });

  it('muestra el estado vacío cuando la búsqueda no encuentra nada', async () => {
    fakeApi({
      ...signedIn(),
      'GET /api/v1/products': (request) => json(200, query(request).get('q') ? page([]) : page([aProduct()])),
    });
    renderApp('/productos');
    await productsTable();

    await userEvent.type(screen.getByRole('searchbox', { name: 'Buscar' }), 'martillo');

    expect(await screen.findByRole('heading', { name: 'No encontramos «martillo»' })).toBeInTheDocument();
    expect(screen.queryByRole('table')).not.toBeInTheDocument();
  });

  it('invita a crear el primer producto cuando el catálogo está vacío', async () => {
    fakeApi({ ...signedIn(), 'GET /api/v1/products': json(200, page([])) });
    renderApp('/productos');

    expect(await screen.findByRole('heading', { name: 'Aún no hay productos' })).toBeInTheDocument();
    expect(screen.getAllByRole('link', { name: 'Nuevo producto' })).not.toHaveLength(0);
  });

  it('un producto con variantes muestra cuántas tiene y el precio desde el menor', async () => {
    const garment = aGarment();
    garment.variants[5]!.active = false; // L, Azul
    fakeApi({ ...signedIn(), 'GET /api/v1/products': json(200, page([garment, aProduct()])) });
    renderApp('/productos');

    const row = (await productsTable()).getByRole('row', { name: /Camiseta básica/ });

    expect(plain(row.textContent)).toContain('6 variantes · und');
    expect(plain(row.textContent)).toContain('Desde $ 29.750');
    // All its variants cost the same and have the same margin.
    expect(plain(row.textContent)).toContain('$ 15.000');
    expect(plain(row.textContent)).toContain('40 %');
    expect(within(row).queryByText('CAM-S-ROJO')).not.toBeInTheDocument();
  });

  it('muestra costo y margen a quien edita el catálogo', async () => {
    fakeApi({ ...signedIn(aUser('admin')), 'GET /api/v1/products': json(200, page([aProduct()])) });
    renderApp('/productos');

    const table = await productsTable();

    expect(table.getByRole('columnheader', { name: 'Costo' })).toBeInTheDocument();
    expect(table.getByRole('columnheader', { name: 'Margen' })).toBeInTheDocument();
    expect(plain(table.getByRole('row', { name: /Tubo PVC/ }).textContent)).toContain('16,67 %');
    expect(table.getByRole('link', { name: 'Editar Tubo PVC presión 1/2 pulgada' })).toHaveAttribute(
      'href',
      '/productos/01920000-0000-7000-8000-000000000101/editar',
    );
  });

  it('el cajero no ve costo, margen, editar ni Nuevo producto', async () => {
    fakeApi({ ...signedIn(aUser('cashier')), 'GET /api/v1/products': json(200, page([withoutCost(aProduct())])) });
    renderApp('/productos');

    const table = await productsTable();

    expect(table.queryByRole('columnheader', { name: 'Costo' })).not.toBeInTheDocument();
    expect(table.queryByRole('columnheader', { name: 'Margen' })).not.toBeInTheDocument();
    expect(table.queryByRole('link', { name: /Editar/ })).not.toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'Nuevo producto' })).not.toBeInTheDocument();
  });

  it('el botón de importar solo aparece con el permiso imports:run', async () => {
    fakeApi({ ...signedIn(), 'GET /api/v1/products': json(200, page([aProduct()])) });
    renderApp('/productos');

    expect(await screen.findByRole('link', { name: 'Importar desde Excel' })).toHaveAttribute(
      'href',
      '/productos/importar',
    );
  });

  it('el bodeguero no ve el botón de importar', async () => {
    fakeApi({ ...signedIn(aUser('warehouse')), 'GET /api/v1/products': json(200, page([aProduct()])) });
    renderApp('/productos');

    await productsTable();

    expect(screen.queryByRole('link', { name: 'Importar desde Excel' })).not.toBeInTheDocument();
  });

  it('muestra un error con la opción de reintentar si la API falla', async () => {
    let attempt = 0;
    fakeApi({
      ...signedIn(),
      'GET /api/v1/products': () =>
        ++attempt === 1
          ? json(500, { error: { code: 'internal', message: 'Error interno' } })
          : json(200, page([aProduct()])),
    });
    renderApp('/productos');

    expect(await screen.findByRole('alert')).toHaveTextContent('No pudimos cargar los productos');
    await userEvent.click(screen.getByRole('button', { name: 'Reintentar' }));

    expect(await productsTable()).toBeTruthy();
  });
});
