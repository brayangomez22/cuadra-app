import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { fakeApi, json, signedIn } from '@/test/fakeApi';
import { aCement, aGarment, aLocation, aProduct, ids, page, plain, query, variantId } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { aStockLevel } from '../../testing/fixtures';

const stockKey = `GET /api/v1/locations/${ids.main}/stock`;

// The catalog finds the cement by name (and a garment by color in a clothing
// store); without a term it answers the names of the rows.
function searchApi() {
  return fakeApi({
    ...signedIn(),
    'GET /api/v1/locations': json(200, { items: [aLocation()] }),
    [stockKey]: json(200, page([aStockLevel({ variant_id: ids.cement, product_id: ids.cement, quantity: '2.5' })])),
    'GET /api/v1/products': (request) => {
      const q = new URL(request.url).searchParams.get('q');
      if (q === 'cemento') return json(200, page([aCement()]));
      if (q === 'azul') return json(200, page([aGarment()]));
      if (q) return json(200, page([]));
      return json(200, page([aProduct(), aCement(), aGarment()]));
    },
  });
}

describe('StockPage: buscar', () => {
  it('busca en el catálogo y pide la existencia de lo que encontró (ferretería)', async () => {
    const api = searchApi();
    const { router } = renderApp('/inventario');

    await userEvent.type(await screen.findByRole('searchbox', { name: 'Buscar' }), 'cemento');

    const table = within(await screen.findByRole('table', { name: 'Existencias en Bodega principal' }));
    expect(plain((await table.findByRole('row', { name: /Cemento gris/ })).textContent)).toContain('2,5 bulto');
    await waitFor(() => expect(query(api.calls(stockKey).at(-1)!).getAll('variant_ids')).toEqual([ids.cement]));
    expect(router.state.location.search).toContain('buscar=cemento');
  });

  it('pide la existencia de todas las variantes que encontró (almacén de ropa)', async () => {
    const api = searchApi();
    renderApp('/inventario?buscar=azul');

    await waitFor(() =>
      expect(query(api.calls(stockKey).at(-1)!).getAll('variant_ids')).toContain(variantId('M', 'Azul')),
    );
    expect(query(api.calls(stockKey).at(-1)!).getAll('variant_ids')).toHaveLength(aGarment().variants.length);
  });

  it('si nada coincide lo dice y no pide la existencia de toda la sede', async () => {
    const api = searchApi();
    renderApp('/inventario?buscar=xyz');

    expect(await screen.findByText('Ningún producto coincide con «xyz»')).toBeInTheDocument();
    expect(api.calls(stockKey)).toHaveLength(0);
  });
});
