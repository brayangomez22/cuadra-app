import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { apiError, json } from '@/test/fakeApi';
import { plain } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { aDistributor, aPurchase, purchaseId, supplierIds } from '../../testing/fixtures';
import { aSupplierPayment } from '../../testing/paymentFixtures';
import { purchasesApi } from '../../testing/purchasesApi';
import { choose, retype } from '../../testing/purchaseSteps';

type Routes = NonNullable<NonNullable<Parameters<typeof purchasesApi>[0]>['routes']>;

function paymentsApi(purchase = aPurchase(), payments = [aSupplierPayment()], extra: Routes = {}) {
  return purchasesApi({
    routes: {
      [`GET /api/v1/purchases/${purchaseId}`]: json(200, purchase),
      [`GET /api/v1/purchases/${purchaseId}/payments`]: json(200, { items: payments }),
      [`GET /api/v1/suppliers/${supplierIds.hardware}`]: json(200, aDistributor()),
      [`POST /api/v1/purchases/${purchaseId}/payments`]: json(201, aSupplierPayment()),
      ...extra,
    },
  });
}

async function openPayment() {
  await userEvent.click(await screen.findByRole('button', { name: 'Registrar pago' }));
  return screen.findByRole('dialog', { name: 'Registrar un pago a Distribuidora Andina SAS' });
}

describe('PurchaseDetailPage: pagos al proveedor', () => {
  it('lista los pagos con su medio y lo que quedó por pagar', async () => {
    paymentsApi(aPurchase({ balance_due: '18480.00' }));
    renderApp(`/compras/${purchaseId}`);

    const table = within(await screen.findByRole('table', { name: 'Pagos al proveedor' }));
    const row = await table.findByRole('row', { name: /Transferencia/ });
    expect(plain(row.textContent)).toContain('27/10/2026');
    expect(plain(row.textContent)).toContain('$ 300.000');
    expect(plain(row.textContent)).toContain('$ 18.480');
  });

  it('registra un pago: propone el saldo y la fecha de hoy, y envía lo escrito', async () => {
    const api = paymentsApi(aPurchase(), []);
    renderApp(`/compras/${purchaseId}`);

    const element = await openPayment();
    const dialog = within(element);
    expect(dialog.getByLabelText('Valor')).toHaveValue('318480');
    expect(dialog.getByLabelText('Fecha del pago')).not.toHaveValue('');
    await choose('Medio de pago', 'Transferencia', element);
    await retype(dialog.getByLabelText('Valor'), '300000');
    await retype(dialog.getByLabelText('Fecha del pago'), '2026-10-27');
    await userEvent.click(dialog.getByRole('button', { name: 'Registrar pago' }));

    expect(await screen.findByText('Pago registrado')).toBeInTheDocument();
    expect(await api.calls(`POST /api/v1/purchases/${purchaseId}/payments`)[0]?.json()).toEqual({
      method: 'transfer',
      amount: '300000',
      paid_on: '2026-10-27',
    });
  });

  it('muestra en el valor que el pago pasa del saldo', async () => {
    paymentsApi(aPurchase(), [], {
      [`POST /api/v1/purchases/${purchaseId}/payments`]: apiError(409, {
        code: 'payment_exceeds_balance',
        message: 'El pago pasa de lo que falta pagar de la compra.',
      }),
    });
    renderApp(`/compras/${purchaseId}`);

    const element = await openPayment();
    const dialog = within(element);
    await choose('Medio de pago', 'Efectivo', element);
    await userEvent.click(dialog.getByRole('button', { name: 'Registrar pago' }));

    await waitFor(() =>
      expect(dialog.getByLabelText('Valor')).toHaveAccessibleDescription(
        expect.stringContaining('El pago pasa de lo que falta pagar'),
      ),
    );
  });

  it('una compra de contado no recibe pagos ni muestra la lista', async () => {
    paymentsApi(aPurchase({ payment: 'cash', status: 'paid', balance_due: '0.00', due_on: undefined }), []);
    renderApp(`/compras/${purchaseId}`);

    expect(await screen.findByRole('button', { name: 'Anular compra' })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Registrar pago' })).not.toBeInTheDocument();
    expect(screen.queryByRole('table', { name: 'Pagos al proveedor' })).not.toBeInTheDocument();
  });
});
