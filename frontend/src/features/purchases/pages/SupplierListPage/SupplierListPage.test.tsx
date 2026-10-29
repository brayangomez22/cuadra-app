import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser } from '@/test/fakeApi';
import { plain, query } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { aDistributor, aTextileSupplier, supplierIds } from '../../testing/fixtures';
import { purchasesApi } from '../../testing/purchasesApi';
import { choose } from '../../testing/purchaseSteps';

const cells = (row: HTMLElement) =>
  within(row)
    .getAllByRole('cell')
    .map((cell) => plain(cell.textContent));

describe('SupplierListPage', () => {
  it('lista los proveedores con su plazo y lo que se le debe', async () => {
    purchasesApi();
    renderApp('/compras/proveedores');

    const table = within(await screen.findByRole('table', { name: 'Proveedores' }));
    expect(cells(table.getByRole('row', { name: /Distribuidora Andina/ }))).toEqual([
      'Distribuidora Andina SAS',
      '800765432-1',
      '30 días',
      'Jorge Peña6017654321',
      '$ 318.480',
      'Compras',
    ]);
    expect(cells(table.getByRole('row', { name: /Confecciones del Valle/ })).slice(2, 5)).toEqual([
      'Contado',
      'ventas@confeccionesvalle.co',
      '$ 0',
    ]);
    expect(table.getByRole('link', { name: 'Compras de Distribuidora Andina SAS' })).toHaveAttribute(
      'href',
      `/compras?proveedor=${supplierIds.hardware}`,
    );
  });

  it('busca por nombre o NIT y filtra por estado', async () => {
    const api = purchasesApi({ suppliers: [aDistributor(), aTextileSupplier({ active: false })] });
    renderApp('/compras/proveedores');

    await userEvent.type(await screen.findByRole('searchbox', { name: 'Buscar' }), '800765');
    await waitFor(() => {
      expect(query(api.calls('GET /api/v1/suppliers').at(-1)!).get('q')).toBe('800765');
    });
    await userEvent.clear(screen.getByRole('searchbox', { name: 'Buscar' }));
    await choose('Estado', 'Inactivos');

    expect(await screen.findByRole('row', { name: /Confecciones del Valle/ })).toBeInTheDocument();
    expect(screen.queryByRole('row', { name: /Distribuidora Andina/ })).not.toBeInTheDocument();
  });

  it('sin purchases:write no ofrece crear ni editar', async () => {
    purchasesApi({ user: { ...aUser('admin'), permissions: ['purchases:read'] } });
    renderApp('/compras/proveedores');

    await screen.findByRole('table', { name: 'Proveedores' });
    expect(screen.queryByRole('link', { name: 'Nuevo proveedor' })).not.toBeInTheDocument();
    expect(screen.queryByRole('link', { name: /Editar/ })).not.toBeInTheDocument();
  });
});
