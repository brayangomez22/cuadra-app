import { screen, within } from '@testing-library/react';
import { json } from '@/test/fakeApi';
import { plain } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { supplierIds } from '../../testing/fixtures';
import { aPayablesPage } from '../../testing/paymentFixtures';
import { purchasesApi } from '../../testing/purchasesApi';

describe('PayablesPage', () => {
  it('suma lo que se debe, lo vencido y lo que vence pronto', async () => {
    purchasesApi({ routes: { 'GET /api/v1/payables': json(200, aPayablesPage()) } });
    renderApp('/compras/por-pagar');

    const summary = within(await screen.findByRole('region', { name: 'Resumen de lo que se debe' }));
    expect(plain(summary.getByText('Se debe en total').nextElementSibling?.textContent)).toBe('$ 726.960');
    expect(plain(summary.getByText('Vencido').nextElementSibling?.textContent)).toBe('$ 318.480');
    expect(plain(summary.getByText('Vence en 7 días').nextElementSibling?.textContent)).toBe('$ 318.480');
  });

  it('muestra cada proveedor y enlaza a sus compras por pagar', async () => {
    purchasesApi({ routes: { 'GET /api/v1/payables': json(200, aPayablesPage()) } });
    renderApp('/compras/por-pagar');

    const table = within(await screen.findByRole('table', { name: 'Lo que se debe a cada proveedor' }));
    const row = table.getByRole('row', { name: /Distribuidora Andina SAS/ });
    expect(plain(row.textContent)).toContain('$ 636.960');
    expect(within(row).getByText('Vencido')).toBeInTheDocument();
    expect(within(row).getByRole('link', { name: 'Distribuidora Andina SAS' })).toHaveAttribute(
      'href',
      `/compras?proveedor=${supplierIds.hardware}&estado=por-pagar`,
    );
  });

  it('sin deudas lo dice', async () => {
    purchasesApi({
      routes: {
        'GET /api/v1/payables': json(200, {
          items: [],
          total: 0,
          totals: { balance: '0.00', overdue: '0.00', due_soon: '0.00', suppliers: 0 },
        }),
      },
    });
    renderApp('/compras/por-pagar');

    expect(await screen.findByText('No se le debe nada a ningún proveedor')).toBeInTheDocument();
  });
});
