import { screen, within } from '@testing-library/react';
import { json } from '@/test/fakeApi';
import { plain } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { aDistributor, aPurchase, purchaseId, supplierIds } from '../../testing/fixtures';
import { purchasesApi } from '../../testing/purchasesApi';

const cells = (row: HTMLElement) =>
  within(row)
    .getAllByRole('cell')
    .map((cell) => plain(cell.textContent));

const routes = {
  [`GET /api/v1/purchases/${purchaseId}`]: json(200, aPurchase()),
  [`GET /api/v1/suppliers/${supplierIds.hardware}`]: json(200, aDistributor()),
};

describe('PurchaseDetailPage', () => {
  it('muestra la factura, el proveedor, la sede y el saldo por pagar', async () => {
    purchasesApi({ routes });
    renderApp(`/compras/${purchaseId}`);

    expect(await screen.findByRole('heading', { name: 'Compra FE 1234' })).toBeInTheDocument();
    const summary = within(screen.getByRole('region', { name: 'Datos de la factura' }));
    expect(await summary.findByText('Distribuidora Andina SAS')).toBeInTheDocument();
    expect(await summary.findByText('Bodega principal')).toBeInTheDocument();
    expect(summary.getByText('Crédito')).toBeInTheDocument();
    expect(summary.getByText('19/11/2026')).toBeInTheDocument();
    const totals = within(screen.getByRole('region', { name: 'Totales de la compra' }));
    expect(plain(totals.getByText('IVA 19 %').nextElementSibling?.textContent)).toBe('$ 36.480');
    expect(plain(totals.getByText('Por pagar').nextElementSibling?.textContent)).toBe('$ 318.480');
  });

  it('muestra las líneas en su unidad y lo que entró en la base', async () => {
    purchasesApi({ routes });
    renderApp(`/compras/${purchaseId}`);

    const table = within(await screen.findByRole('table', { name: 'Líneas de la compra' }));
    // The first cell repeats quantity, cost and IVA for a phone, where their columns are hidden.
    expect(cells(table.getByRole('row', { name: /Tubo PVC/ }))).toEqual([
      'Tubo PVC presión 1/2 pulgada2 caja × $ 96.000 · IVA 19 %',
      '2 caja',
      '24 (caja de 12)',
      '$ 96.000',
      '19 %',
      '$ 228.480',
    ]);
    expect(cells(table.getByRole('row', { name: /Camiseta/ })).slice(1, 3)).toEqual(['6 und', '—']);
  });

  it('sin líneas en otra unidad no muestra la columna de lo que entró', async () => {
    const purchase = aPurchase();
    const lines = purchase.lines.filter((line) => !line.unit_id);
    purchasesApi({ routes: { ...routes, [`GET /api/v1/purchases/${purchaseId}`]: json(200, { ...purchase, lines }) } });
    renderApp(`/compras/${purchaseId}`);

    const table = within(await screen.findByRole('table', { name: 'Líneas de la compra' }));
    expect(table.getAllByRole('columnheader').map((th) => th.textContent)).toEqual([
      'Producto',
      'Cantidad',
      'Costo unitario',
      'IVA',
      'Total',
    ]);
  });
});
