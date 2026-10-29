import { screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { json } from '@/test/fakeApi';
import { ids, plain } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { aDistributor, aPurchase, purchaseId, supplierIds } from '../../testing/fixtures';
import { purchasesApi } from '../../testing/purchasesApi';
import {
  choose,
  fillAmounts,
  fillInvoice,
  lineGroup,
  pickProduct,
  registerPurchase,
} from '../../testing/purchaseSteps';

const detailRoutes = {
  [`GET /api/v1/purchases/${purchaseId}`]: json(200, aPurchase()),
  [`GET /api/v1/suppliers/${supplierIds.hardware}`]: json(200, aDistributor()),
};

describe('PurchaseFormPage: registrar', () => {
  it('registra una compra a crédito con el vencimiento propuesto por el plazo del proveedor', async () => {
    const api = purchasesApi({ routes: detailRoutes });
    renderApp('/compras/nueva');

    await fillInvoice();
    await choose('Forma de pago', 'Crédito');
    expect(screen.getByLabelText('Vencimiento')).toHaveValue('2026-11-19');
    expect(screen.getByLabelText('Vencimiento')).toHaveAccessibleDescription(/30 días/);
    await pickProduct(1, 'cemento', 'Cemento gris 50kg');
    await fillAmounts(1, '10', '28000');
    await registerPurchase();

    expect(await screen.findByRole('heading', { name: 'Compra FE 1234' })).toBeInTheDocument();
    expect(await api.calls('POST /api/v1/purchases')[0]?.json()).toEqual({
      supplier_id: supplierIds.hardware,
      location_id: ids.main,
      invoice_number: 'FE 1234',
      issued_on: '2026-10-20',
      payment: 'credit',
      due_on: '2026-11-19',
      lines: [{ variant_id: ids.cement, quantity: '10', unit_cost: '28000', tax_rate: '0.19' }],
    });
  });

  it('de contado no pide vencimiento', async () => {
    purchasesApi();
    renderApp('/compras/nueva');

    await fillInvoice();

    expect(screen.getByRole('combobox', { name: 'Forma de pago' })).toHaveTextContent('Contado');
    expect(screen.queryByLabelText('Vencimiento')).not.toBeInTheDocument();
  });

  it('muestra el subtotal, el IVA por tasa y el total mientras se escribe', async () => {
    purchasesApi();
    renderApp('/compras/nueva');

    await pickProduct(1, 'cemento', 'Cemento gris 50kg');
    await fillAmounts(1, '10', '28000');
    await userEvent.click(screen.getByRole('button', { name: 'Agregar línea' }));
    await pickProduct(2, 'tubo', 'Tubo PVC');
    await choose('IVA', '5\u00a0%', lineGroup(2));
    await fillAmounts(2, '4', '2500');

    const totals = within(screen.getByRole('region', { name: 'Totales de la compra' }));
    expect(plain(totals.getByText('Subtotal').nextElementSibling?.textContent)).toBe('$ 290.000');
    expect(plain(totals.getByText('IVA 5 %').nextElementSibling?.textContent)).toBe('$ 500');
    expect(plain(totals.getByText('IVA 19 %').nextElementSibling?.textContent)).toBe('$ 53.200');
    expect(plain(totals.getByText('Total').nextElementSibling?.textContent)).toBe('$ 343.700');
  });

  it('quita una línea', async () => {
    purchasesApi();
    renderApp('/compras/nueva');

    await screen.findByRole('group', { name: 'Línea 1' });
    await userEvent.click(screen.getByRole('button', { name: 'Agregar línea' }));
    await userEvent.click(screen.getByRole('button', { name: 'Quitar la línea 2' }));

    expect(screen.queryByRole('group', { name: 'Línea 2' })).not.toBeInTheDocument();
  });
});
