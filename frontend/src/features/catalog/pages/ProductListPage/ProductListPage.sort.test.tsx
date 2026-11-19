import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser, fakeApi, json, signedIn } from '@/test/fakeApi';
import { aCement, aGarment, page } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';

const productsKey = 'GET /api/v1/products';
const categories = {
  items: [{ id: '01920000-0000-7000-8000-000000000701', name: 'Camisetas', created_at: '2026-10-01T15:00:00Z' }],
};

function catalogApi(total = 2, me = aUser()) {
  return fakeApi({
    ...signedIn(me),
    [productsKey]: json(200, { ...page([aCement(), aGarment()]), total }),
    'GET /api/v1/categories': json(200, categories),
  });
}

const lastQuery = (api: ReturnType<typeof catalogApi>) => new URL(api.calls(productsKey).at(-1)!.url).searchParams;

describe('ProductListPage: orden, categoría y páginas', () => {
  it('ordena por margen, el más bajo primero, y filtra por categoría', async () => {
    const api = catalogApi();
    renderApp('/productos');

    await userEvent.click(await screen.findByRole('combobox', { name: 'Ordenar por' }));
    await userEvent.click(await screen.findByRole('option', { name: 'Margen, menor primero' }));
    await userEvent.click(screen.getByRole('combobox', { name: 'Categoría' }));
    await userEvent.click(await screen.findByRole('option', { name: 'Camisetas' }));

    await waitFor(() => {
      expect(lastQuery(api).get('sort')).toBe('margin');
      expect(lastQuery(api).get('category_id')).toBe(categories.items[0]!.id);
    });
  });

  it('el cajero, que no ve márgenes, ordena por nombre o precio', async () => {
    catalogApi(2, aUser('cashier'));
    renderApp('/productos');

    await userEvent.click(await screen.findByRole('combobox', { name: 'Ordenar por' }));
    expect(await screen.findByRole('option', { name: 'Precio, menor primero' })).toBeInTheDocument();
    expect(screen.queryByRole('option', { name: 'Margen, menor primero' })).not.toBeInTheDocument();
  });

  it('con muchas páginas salta directo a una', async () => {
    const api = catalogApi(2000);
    renderApp('/productos');

    await userEvent.type(await screen.findByRole('spinbutton', { name: /Página/ }), '37');
    await userEvent.click(screen.getByRole('button', { name: 'Ir' }));

    await waitFor(() => {
      expect(lastQuery(api).get('offset')).toBe(String(36 * 20));
    });
  });
});
