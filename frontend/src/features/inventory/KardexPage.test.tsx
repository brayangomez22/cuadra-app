import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser, fakeApi, json, signedIn } from '@/test/fakeApi';
import { aLocation, aMovement, aProduct, ids, page, plain, query, withoutCost, withoutCosts } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';


const movements = [
  aMovement({
    id: '01920000-0000-7000-8000-000000000403',
    type: 'adjustment',
    quantity: '-2.5',
    balance_after: '7.5',
    unit_cost: '8000.00',
    reason: 'Tubo roto en la bodega',
    occurred_at: '2026-10-11T03:30:00Z',
  }),
  aMovement(),
];

function kardexApi(extra: Parameters<typeof fakeApi>[0] = {}) {
  return fakeApi({
    ...signedIn(),
    [`GET /api/v1/products/${ids.pipe}`]: json(200, aProduct()),
    'GET /api/v1/locations': json(200, { items: [aLocation(), aLocation({ id: ids.north, name: 'Sede norte' })] }),
    [`GET /api/v1/products/${ids.pipe}/kardex`]: json(200, page(movements)),
    ...extra,
  });
}

async function kardexTable() {
  return within(await screen.findByRole('table', { name: 'Movimientos de Tubo PVC presión 1/2 pulgada' }));
}

describe('KardexPage', () => {
  it('lista los movimientos con su signo, su saldo y la hora de Bogotá', async () => {
    kardexApi();
    renderApp(`/productos/${ids.pipe}/kardex`);

    expect(await screen.findByRole('heading', { level: 1, name: 'Tubo PVC presión 1/2 pulgada' })).toBeInTheDocument();
    const table = await kardexTable();
    const adjustment = await table.findByRole('row', { name: /Ajuste/ });
    const receipt = table.getByRole('row', { name: /Entrada de mercancía/ });

    expect(plain(adjustment.textContent)).toContain('10/10/2026, 10:30 p. m.');
    expect(plain(adjustment.textContent)).toContain('-2,5');
    expect(plain(adjustment.textContent)).toContain('7,5');
    expect(adjustment).toHaveTextContent('Tubo roto en la bodega');
    expect(adjustment).toHaveTextContent('Bodega principal');
    expect(plain(receipt.textContent)).toContain('+10');
    expect(plain(receipt.textContent)).toContain('$ 8.000');
  });

  it('filtra por sede y la conserva en la URL', async () => {
    const api = kardexApi();
    const { router } = renderApp(`/productos/${ids.pipe}/kardex`);
    await kardexTable();

    await userEvent.click(screen.getByRole('combobox', { name: 'Sede' }));
    await userEvent.click(await screen.findByRole('option', { name: 'Sede norte' }));

    await waitFor(() =>
      expect(query(api.calls(`GET /api/v1/products/${ids.pipe}/kardex`).at(-1)!).get('location_id')).toBe(ids.north),
    );
    expect(router.state.location.search).toBe(`?sede=${ids.north}`);
  });

  it('llega filtrado por la sede de la URL', async () => {
    const api = kardexApi();
    renderApp(`/productos/${ids.pipe}/kardex?sede=${ids.north}`);
    await kardexTable();

    const [first] = api.calls(`GET /api/v1/products/${ids.pipe}/kardex`);
    expect(query(first!).get('location_id')).toBe(ids.north);
  });

  it('oculta el costo a quien no edita el catálogo', async () => {
    kardexApi({
      ...signedIn(aUser('warehouse')),
      [`GET /api/v1/products/${ids.pipe}`]: json(200, withoutCost(aProduct())),
      [`GET /api/v1/products/${ids.pipe}/kardex`]: json(
        200,
        page(movements.map(withoutCosts)),
      ),
    });
    renderApp(`/productos/${ids.pipe}/kardex`);

    const table = await kardexTable();

    expect(table.queryByRole('columnheader', { name: 'Costo unitario' })).not.toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Registrar entrada' })).toHaveAttribute(
      'href',
      `/inventario/entrada?producto=${ids.pipe}`,
    );
  });

  it('avisa cuando el producto aún no tiene movimientos', async () => {
    kardexApi({ [`GET /api/v1/products/${ids.pipe}/kardex`]: json(200, page([])) });
    renderApp(`/productos/${ids.pipe}/kardex`);

    expect(await screen.findByRole('heading', { name: 'Este producto aún no tiene movimientos' })).toBeInTheDocument();
  });

  it('el cajero no ve Registrar entrada', async () => {
    kardexApi({ ...signedIn(aUser('cashier')) });
    renderApp(`/productos/${ids.pipe}/kardex`);

    await kardexTable();

    expect(screen.queryByRole('link', { name: 'Registrar entrada' })).not.toBeInTheDocument();
  });
});
