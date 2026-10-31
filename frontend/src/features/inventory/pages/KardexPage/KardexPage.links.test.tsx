import { screen, within } from '@testing-library/react';
import { aUser, fakeApi, json, signedIn } from '@/test/fakeApi';
import { aLocation, aProduct, ids, page } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { aMovement } from '../../testing/fixtures';

const purchaseId = '01920000-0000-7000-8000-000000000b01';
const saleId = '01920000-0000-7000-8000-000000000501';

function kardexApi(extra: Parameters<typeof fakeApi>[0] = {}) {
  return fakeApi({
    ...signedIn(),
    [`GET /api/v1/products/${ids.pipe}`]: json(200, aProduct()),
    'GET /api/v1/locations': json(200, { items: [aLocation(), aLocation({ id: ids.north, name: 'Sede norte' })] }),
    [`GET /api/v1/variants/${ids.pipe}/kardex`]: json(200, page([aMovement()])),
    ...extra,
  });
}

async function kardexTable() {
  return within(await screen.findByRole('table', { name: 'Movimientos de Tubo PVC presión 1/2 pulgada' }));
}

describe('KardexPage: documentos', () => {
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

  it.each([
    { type: 'sale_out', label: 'Venta' },
    { type: 'sale_void', label: 'Venta anulada' },
  ] as const)('enlaza «$label» con su venta', async ({ type, label }) => {
    kardexApi({
      [`GET /api/v1/variants/${ids.pipe}/kardex`]: json(
        200,
        page([aMovement({ type, reference: { kind: 'sale', id: saleId } })]),
      ),
    });
    renderApp(`/productos/${ids.pipe}/kardex`);

    const table = await kardexTable();
    expect(await table.findByRole('link', { name: label })).toHaveAttribute('href', `/ventas/${saleId}`);
  });

  it('sin permiso de ver ventas la salida no enlaza', async () => {
    kardexApi({
      ...signedIn(aUser('warehouse')),
      [`GET /api/v1/variants/${ids.pipe}/kardex`]: json(
        200,
        page([aMovement({ type: 'sale_out', reference: { kind: 'sale', id: saleId } })]),
      ),
    });
    renderApp(`/productos/${ids.pipe}/kardex`);

    const table = await kardexTable();
    expect(await table.findByText('Venta')).toBeInTheDocument();
    expect(table.queryByRole('link', { name: 'Venta' })).not.toBeInTheDocument();
  });
});
