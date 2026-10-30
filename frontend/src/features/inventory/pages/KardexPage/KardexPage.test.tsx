import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser, fakeApi, json, signedIn } from '@/test/fakeApi';
import { aGarment, aLocation, aProduct, ids, page, plain, query, variantId, withoutCost } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { aMovement, withoutCosts } from '../../testing/fixtures';

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
    [`GET /api/v1/variants/${ids.pipe}/kardex`]: json(200, page(movements)),
    ...extra,
  });
}

async function kardexTable() {
  return within(await screen.findByRole('table', { name: 'Movimientos de Tubo PVC presión 1/2 pulgada' }));
}

const purchaseId = '01920000-0000-7000-8000-000000000b01';

describe('KardexPage', () => {
  it('enlaza la entrada de una compra con su factura', async () => {
    kardexApi({
      [`GET /api/v1/variants/${ids.pipe}/kardex`]: json(
        200,
        page([aMovement({ reference: { kind: 'purchase', id: purchaseId } })]),
      ),
    });
    renderApp(`/productos/${ids.pipe}/kardex`);

    const table = await kardexTable();
    expect(await table.findByRole('link', { name: 'Entrada de mercancía' })).toHaveAttribute(
      'href',
      `/compras/${purchaseId}`,
    );
  });

  it('sin permiso de ver compras la entrada no enlaza', async () => {
    kardexApi({
      ...signedIn(aUser('cashier')),
      [`GET /api/v1/variants/${ids.pipe}/kardex`]: json(
        200,
        page([aMovement({ reference: { kind: 'purchase', id: purchaseId } })]),
      ),
    });
    renderApp(`/productos/${ids.pipe}/kardex`);

    const table = await kardexTable();
    expect(await table.findByText('Entrada de mercancía')).toBeInTheDocument();
    expect(table.queryByRole('link', { name: 'Entrada de mercancía' })).not.toBeInTheDocument();
  });

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

  it('con todas las sedes, cada saldo dice de qué sede es', async () => {
    kardexApi({
      [`GET /api/v1/variants/${ids.pipe}/kardex`]: json(
        200,
        page([aMovement({ location_id: ids.north, balance_after: '4' }), aMovement({ id: ids.cashSession })]),
      ),
    });
    renderApp(`/productos/${ids.pipe}/kardex`);

    const table = await kardexTable();
    expect(table.getByRole('columnheader', { name: 'Saldo en su sede' })).toBeInTheDocument();
    const [north, main] = table.getAllByRole('row', { name: /Entrada de mercancía/ });
    expect(plain(within(north!).getAllByRole('cell')[3]?.textContent)).toBe('4en Sede norte');
    expect(plain(within(main!).getAllByRole('cell')[3]?.textContent)).toBe('10en Bodega principal');
  });

  it('en una sede, el saldo es el de esa sede', async () => {
    kardexApi();
    renderApp(`/productos/${ids.pipe}/kardex?sede=${ids.main}`);

    const table = await kardexTable();
    expect(table.getByRole('columnheader', { name: 'Saldo en la sede' })).toBeInTheDocument();
    const receipt = await table.findByRole('row', { name: /Entrada de mercancía/ });
    expect(plain(within(receipt).getAllByRole('cell')[3]?.textContent)).toBe('10');
  });

  it('filtra por sede y la conserva en la URL', async () => {
    const api = kardexApi();
    const { router } = renderApp(`/productos/${ids.pipe}/kardex`);
    await kardexTable();

    await userEvent.click(screen.getByRole('combobox', { name: 'Sede' }));
    await userEvent.click(await screen.findByRole('option', { name: 'Sede norte' }));

    await waitFor(() =>
      expect(query(api.calls(`GET /api/v1/variants/${ids.pipe}/kardex`).at(-1)!).get('location_id')).toBe(ids.north),
    );
    expect(router.state.location.search).toBe(`?sede=${ids.north}`);
  });

  it('llega filtrado por la sede de la URL', async () => {
    const api = kardexApi();
    renderApp(`/productos/${ids.pipe}/kardex?sede=${ids.north}`);
    await kardexTable();

    const [first] = api.calls(`GET /api/v1/variants/${ids.pipe}/kardex`);
    expect(query(first!).get('location_id')).toBe(ids.north);
  });

  it('oculta el costo a quien no edita el catálogo', async () => {
    kardexApi({
      ...signedIn(aUser('warehouse')),
      [`GET /api/v1/products/${ids.pipe}`]: json(200, withoutCost(aProduct())),
      [`GET /api/v1/variants/${ids.pipe}/kardex`]: json(200, page(movements.map(withoutCosts))),
    });
    renderApp(`/productos/${ids.pipe}/kardex`);

    const table = await kardexTable();

    expect(table.queryByRole('columnheader', { name: 'Costo unitario' })).not.toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Registrar entrada' })).toHaveAttribute(
      'href',
      `/inventario/entrada?producto=${ids.pipe}&variante=${ids.pipe}`,
    );
  });

  it('avisa cuando el producto aún no tiene movimientos', async () => {
    kardexApi({ [`GET /api/v1/variants/${ids.pipe}/kardex`]: json(200, page([])) });
    renderApp(`/productos/${ids.pipe}/kardex`);

    expect(await screen.findByRole('heading', { name: 'Este producto aún no tiene movimientos' })).toBeInTheDocument();
  });

  it('el kardex de un producto simple consulta el de su variante por defecto', async () => {
    const api = kardexApi();
    renderApp(`/productos/${ids.pipe}/kardex`);

    await kardexTable();

    expect(api.calls(`GET /api/v1/variants/${ids.pipe}/kardex`)).toHaveLength(1);
    expect(screen.queryByRole('combobox', { name: 'Variante' })).not.toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Registrar entrada' })).toHaveAttribute(
      'href',
      `/inventario/entrada?producto=${ids.pipe}&variante=${ids.pipe}`,
    );
  });

  describe('variantes', () => {
    function garmentApi() {
      return kardexApi({
        [`GET /api/v1/products/${ids.garment}`]: json(200, aGarment()),
        [`GET /api/v1/variants/${variantId('M', 'Azul')}/kardex`]: json(
          200,
          page([aMovement({ variant_id: variantId('M', 'Azul') })]),
        ),
        [`GET /api/v1/variants/${variantId('L', 'Rojo')}/kardex`]: json(200, page([])),
      });
    }

    it('muestra el kardex de la variante de la URL', async () => {
      const api = garmentApi();
      renderApp(`/productos/${ids.garment}/kardex?variante=${variantId('M', 'Azul')}&sede=${ids.main}`);

      expect(await screen.findByRole('heading', { level: 1, name: 'Camiseta básica (Azul · M)' })).toBeInTheDocument();
      await screen.findByRole('table', { name: 'Movimientos de Camiseta básica (Azul · M)' });
      expect(screen.getByRole('combobox', { name: 'Variante' })).toHaveTextContent('Azul · M');
      expect(query(api.calls(`GET /api/v1/variants/${variantId('M', 'Azul')}/kardex`)[0]!).get('location_id')).toBe(
        ids.main,
      );
      expect(screen.getByRole('link', { name: 'Registrar entrada' })).toHaveAttribute(
        'href',
        `/inventario/entrada?producto=${ids.garment}&variante=${variantId('M', 'Azul')}&sede=${ids.main}`,
      );
    });

    it('sin variante en la URL muestra la variante por defecto', async () => {
      const api = kardexApi({
        [`GET /api/v1/products/${ids.garment}`]: json(200, aGarment()),
        [`GET /api/v1/variants/${variantId('S', 'Rojo')}/kardex`]: json(200, page([])),
      });
      renderApp(`/productos/${ids.garment}/kardex`);

      expect(await screen.findByRole('heading', { level: 1, name: 'Camiseta básica (Rojo · S)' })).toBeInTheDocument();
      await waitFor(() => expect(api.calls(`GET /api/v1/variants/${variantId('S', 'Rojo')}/kardex`)).toHaveLength(1));
      expect(api.calls(`GET /api/v1/products/${ids.garment}/kardex`)).toHaveLength(0);
    });

    it('cambia de variante y la conserva en la URL', async () => {
      const api = garmentApi();
      const { router } = renderApp(`/productos/${ids.garment}/kardex?variante=${variantId('M', 'Azul')}`);
      await screen.findByRole('table', { name: 'Movimientos de Camiseta básica (Azul · M)' });

      await userEvent.click(screen.getByRole('combobox', { name: 'Variante' }));
      await userEvent.click(await screen.findByRole('option', { name: 'Rojo · L' }));

      expect(
        await screen.findByRole('heading', { name: 'Esta variante aún no tiene movimientos' }),
      ).toBeInTheDocument();
      expect(api.calls(`GET /api/v1/variants/${variantId('L', 'Rojo')}/kardex`)).toHaveLength(1);
      expect(router.state.location.search).toBe(`?variante=${variantId('L', 'Rojo')}`);
    });
  });

  it('el cajero no ve Registrar entrada', async () => {
    kardexApi({ ...signedIn(aUser('cashier')) });
    renderApp(`/productos/${ids.pipe}/kardex`);

    await kardexTable();

    expect(screen.queryByRole('link', { name: 'Registrar entrada' })).not.toBeInTheDocument();
  });
});
