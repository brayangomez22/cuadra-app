import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser, apiError, fakeApi, json, signedIn } from '@/test/fakeApi';
import { aCement, aLocation, aProduct, ids, page, plain } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { aStockLevel } from '../../testing/fixtures';
import { aReorderRule } from '../../testing/reorderApi';

const rulePath = `/api/v1/locations/${ids.main}/reorder-rules/${ids.cement}`;

function stockApi(user = aUser(), extra: Parameters<typeof fakeApi>[0] = {}) {
  return fakeApi({
    ...signedIn(user),
    'GET /api/v1/locations': json(200, { items: [aLocation()] }),
    [`GET /api/v1/locations/${ids.main}/stock`]: json(
      200,
      page([
        aStockLevel({ quantity: '3', reorder: { min_quantity: '10', max_quantity: '50', needs_reorder: true } }),
        aStockLevel({ variant_id: ids.cement, product_id: ids.cement, quantity: '2.5' }),
      ]),
    ),
    'GET /api/v1/products': json(200, page([aProduct(), aCement()])),
    ...extra,
  });
}

async function rowOf(name: RegExp) {
  const table = within(await screen.findByRole('table', { name: 'Existencias en Bodega principal' }));
  return table.findByRole('row', { name });
}

describe('StockPage: stock mínimo y máximo', () => {
  it('marca la variante por reponer y muestra su mínimo y máximo', async () => {
    stockApi();
    renderApp('/inventario');

    const pipe = await rowOf(/Tubo PVC/);
    expect(within(pipe).getByText('Por reponer')).toBeInTheDocument();
    expect(plain(pipe.textContent)).toContain('10 – 50');
    const cement = await rowOf(/Cemento gris/);
    expect(within(cement).queryByText('Por reponer')).not.toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Por reponer' })).toHaveAttribute(
      'href',
      `/inventario/reponer?sede=${ids.main}`,
    );
  });

  it('define el mínimo y el máximo de una variante con coma decimal', async () => {
    const api = stockApi(aUser('warehouse'), { [`PUT ${rulePath}`]: json(200, aReorderRule('5', '12.5', ids.cement)) });
    renderApp('/inventario');

    const cement = await rowOf(/Cemento gris/);
    await userEvent.click(within(cement).getByRole('button', { name: 'Mínimo y máximo de Cemento gris 50kg' }));
    const dialog = within(await screen.findByRole('dialog', { name: 'Mínimo y máximo' }));
    await userEvent.type(dialog.getByLabelText('Stock mínimo'), '5');
    await userEvent.type(dialog.getByLabelText('Stock máximo'), '12,5');
    await userEvent.click(dialog.getByRole('button', { name: 'Guardar' }));

    await waitFor(() => expect(api.calls(`PUT ${rulePath}`)).toHaveLength(1));
    expect(await api.calls(`PUT ${rulePath}`)[0]!.json()).toEqual({ min_quantity: '5', max_quantity: '12.5' });
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    // The stock is read again so the row shows the new values.
    expect(api.calls(`GET /api/v1/locations/${ids.main}/stock`).length).toBeGreaterThan(1);
  });

  it('lleva el error de la API al campo', async () => {
    stockApi(aUser(), {
      [`PUT ${rulePath}`]: apiError(422, { code: 'invalid_reorder_range', message: 'El stock mínimo no puede…' }),
    });
    renderApp('/inventario');

    await userEvent.click(within(await rowOf(/Cemento gris/)).getByRole('button', { name: /Mínimo y máximo/ }));
    const dialog = within(await screen.findByRole('dialog'));
    await userEvent.type(dialog.getByLabelText('Stock mínimo'), '1');
    await userEvent.type(dialog.getByLabelText('Stock máximo'), '2');
    await userEvent.click(dialog.getByRole('button', { name: 'Guardar' }));

    expect(await dialog.findByText('El stock mínimo no puede…')).toBeInTheDocument();
  });

  it('quita el mínimo y el máximo de una variante que los tiene', async () => {
    const pipePath = `/api/v1/locations/${ids.main}/reorder-rules/${ids.pipe}`;
    const api = stockApi(aUser(), { [`DELETE ${pipePath}`]: new Response(null, { status: 204 }) });
    renderApp('/inventario');

    await userEvent.click(within(await rowOf(/Tubo PVC/)).getByRole('button', { name: /Mínimo y máximo/ }));
    const dialog = within(await screen.findByRole('dialog'));
    expect(dialog.getByLabelText('Stock mínimo')).toHaveValue('10');
    await userEvent.click(dialog.getByRole('button', { name: 'Quitar' }));

    await waitFor(() => expect(api.calls(`DELETE ${pipePath}`)).toHaveLength(1));
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
  });

  it('el cajero ve el mínimo y el máximo pero no los edita', async () => {
    stockApi(aUser('cashier'));
    renderApp('/inventario');

    const pipe = await rowOf(/Tubo PVC/);
    expect(plain(pipe.textContent)).toContain('10 – 50');
    expect(within(pipe).queryByRole('button')).not.toBeInTheDocument();
  });
});
