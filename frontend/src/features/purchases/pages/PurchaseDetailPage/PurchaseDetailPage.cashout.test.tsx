import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { json } from '@/test/fakeApi';
import { ids } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { aDistributor, aPurchase, purchaseId, supplierIds } from '../../testing/fixtures';
import { aSupplierPayment } from '../../testing/paymentFixtures';
import { purchasesApi } from '../../testing/purchasesApi';
import { choose } from '../../testing/purchaseSteps';

const PAY_KEY = `POST /api/v1/purchases/${purchaseId}/payments` as const;
const MOVE_KEY = `POST /api/v1/cash-sessions/${ids.cashSession}/movements` as const;
const myDrawer = {
  id: ids.cashSession,
  location_id: ids.main,
  user_id: '01920000-0000-7000-8000-000000000001',
  status: 'open',
  opening_float: '100000.00',
  opened_at: '2026-10-27T13:00:00Z',
  movements: [],
} as const;

// null: the user has no drawer open.
function cashOutApi(session: typeof myDrawer | null = myDrawer) {
  return purchasesApi({
    routes: {
      [`GET /api/v1/purchases/${purchaseId}`]: json(200, aPurchase()),
      [`GET /api/v1/purchases/${purchaseId}/payments`]: json(200, { items: [] }),
      [`GET /api/v1/suppliers/${supplierIds.hardware}`]: json(200, aDistributor()),
      'GET /api/v1/cash-sessions/current': json(200, { required: true, session: session ?? undefined }),
      [PAY_KEY]: json(201, aSupplierPayment()),
      [MOVE_KEY]: json(201, {
        id: ids.cashSession,
        kind: 'out',
        amount: '318480.00',
        reason: 'x',
        user_id: myDrawer.user_id,
        created_at: '2026-10-27T15:00:00Z',
      }),
    },
  });
}

async function openPayment() {
  await userEvent.click(await screen.findByRole('button', { name: 'Registrar pago' }));
  return screen.findByRole('dialog', { name: 'Registrar un pago a Distribuidora Andina SAS' });
}

describe('PurchaseDetailPage: el pago en efectivo sale de la caja abierta', () => {
  it('en efectivo y con la caja abierta, saca el valor de la caja como egreso', async () => {
    const api = cashOutApi();
    renderApp(`/compras/${purchaseId}`);

    const element = await openPayment();
    await choose('Medio de pago', 'Efectivo', element);
    expect(
      await within(element).findByRole('checkbox', { name: /Sacar \$\s318\.480 de mi caja abierta/ }),
    ).toBeChecked();
    await userEvent.click(within(element).getByRole('button', { name: 'Registrar pago' }));

    await waitFor(() => expect(api.calls(MOVE_KEY)).toHaveLength(1));
    expect(await api.calls(MOVE_KEY)[0]!.json()).toEqual({
      kind: 'out',
      amount: '318480',
      reason: 'Pago a Distribuidora Andina SAS, factura FE 1234',
    });
  });

  it('con otro medio no ofrece sacarlo de la caja', async () => {
    cashOutApi();
    renderApp(`/compras/${purchaseId}`);

    const element = await openPayment();
    await choose('Medio de pago', 'Transferencia', element);
    expect(within(element).queryByRole('checkbox')).not.toBeInTheDocument();
  });

  it('sin caja abierta solo avisa que lo registre en la caja', async () => {
    const api = cashOutApi(null);
    renderApp(`/compras/${purchaseId}`);

    const element = await openPayment();
    await choose('Medio de pago', 'Efectivo', element);
    expect(within(element).getByText(/regístrelo como egreso en la caja/)).toBeInTheDocument();
    await userEvent.click(within(element).getByRole('button', { name: 'Registrar pago' }));

    await waitFor(() => expect(api.calls(PAY_KEY)).toHaveLength(1));
    expect(api.calls(MOVE_KEY)).toHaveLength(0);
  });
});
