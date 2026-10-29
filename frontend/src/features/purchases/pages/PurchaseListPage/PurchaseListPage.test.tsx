import { screen, waitFor, within } from '@testing-library/react';
import { json } from '@/test/fakeApi';
import { page, plain, query } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { aPurchaseSummary, purchaseId, supplierIds } from '../../testing/fixtures';
import { purchasesApi } from '../../testing/purchasesApi';
import { choose } from '../../testing/purchaseSteps';

const cells = (row: HTMLElement) =>
  within(row)
    .getAllByRole('cell')
    .map((cell) => plain(cell.textContent));

const paidInCash = aPurchaseSummary({
  id: '01920000-0000-7000-8000-000000000b02',
  supplier_id: supplierIds.textiles,
  supplier_name: 'Confecciones del Valle',
  invoice_number: 'CV-88',
  issued_on: '2026-10-15',
  due_on: undefined,
  payment: 'cash',
  status: 'paid',
  total: '90000.00',
  balance_due: '0.00',
});

describe('PurchaseListPage', () => {
  it('muestra cada compra con su saldo por pagar y enlaza a su detalle', async () => {
    purchasesApi({ routes: { 'GET /api/v1/purchases': json(200, page([aPurchaseSummary(), paidInCash])) } });
    renderApp('/compras');

    const table = within(await screen.findByRole('table', { name: 'Compras' }));
    expect(cells(table.getByRole('row', { name: /FE 1234/ }))).toEqual([
      'FE 1234',
      'Distribuidora Andina SAS',
      '20/10/2026',
      '19/11/2026',
      '$ 318.480',
      '$ 318.480Por pagar',
    ]);
    expect(cells(table.getByRole('row', { name: /CV-88/ })).slice(3)).toEqual(['Contado', '$ 90.000', 'Pagada']);
    expect(table.getByRole('link', { name: 'FE 1234' })).toHaveAttribute('href', `/compras/${purchaseId}`);
  });

  it('filtra por proveedor desde la URL y por estado', async () => {
    const api = purchasesApi();
    renderApp(`/compras?proveedor=${supplierIds.hardware}`);

    await screen.findByRole('table', { name: 'Compras' });
    expect(query(api.calls('GET /api/v1/purchases')[0]!).get('supplier_id')).toBe(supplierIds.hardware);
    await choose('Estado', 'Por pagar');

    await waitFor(() => {
      expect(query(api.calls('GET /api/v1/purchases').at(-1)!).get('status')).toBe('open');
    });
    expect(query(api.calls('GET /api/v1/purchases').at(-1)!).get('supplier_id')).toBe(supplierIds.hardware);
  });

  it('sin compras invita a registrar la primera', async () => {
    purchasesApi({ routes: { 'GET /api/v1/purchases': json(200, page([])) } });
    renderApp('/compras');

    expect(await screen.findByText('Todavía no hay compras')).toBeInTheDocument();
    expect(screen.getAllByRole('link', { name: /Registrar.*compra/ }).length).toBeGreaterThan(0);
  });
});
