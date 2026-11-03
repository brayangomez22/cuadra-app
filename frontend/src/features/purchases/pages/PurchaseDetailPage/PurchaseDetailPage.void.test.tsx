import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { apiError, json } from '@/test/fakeApi';
import { renderApp } from '@/test/renderApp';
import { aDistributor, aPurchase, purchaseId, supplierIds } from '../../testing/fixtures';
import { purchasesApi } from '../../testing/purchasesApi';
import { write } from '../../testing/purchaseSteps';

const voided = aPurchase({
  status: 'voided',
  balance_due: '0.00',
  voided_at: '2026-10-28T15:00:00Z',
  voided_by: '01920000-0000-7000-8000-000000000001',
  void_reason: 'La factura se registró dos veces',
});

type Routes = NonNullable<NonNullable<Parameters<typeof purchasesApi>[0]>['routes']>;

function voidApi(purchase = aPurchase(), extra: Routes = {}) {
  // Like the API: once voided, the purchase reads voided.
  let current = purchase;
  return purchasesApi({
    routes: {
      [`GET /api/v1/purchases/${purchaseId}`]: () => json(200, current),
      [`GET /api/v1/purchases/${purchaseId}/payments`]: json(200, { items: [] }),
      [`GET /api/v1/suppliers/${supplierIds.hardware}`]: json(200, aDistributor()),
      [`POST /api/v1/purchases/${purchaseId}/void`]: () => {
        current = voided;
        return json(200, voided);
      },
      ...extra,
    },
  });
}

async function openVoid() {
  await userEvent.click(await screen.findByRole('button', { name: 'Anular compra' }));
  return within(await screen.findByRole('dialog', { name: 'Anular la compra FE 1234' }));
}

describe('PurchaseDetailPage: anular', () => {
  it('anula con el motivo, dice que saca el stock y muestra la compra anulada', async () => {
    const api = voidApi();
    renderApp(`/compras/${purchaseId}`);

    const dialog = await openVoid();
    expect(dialog.getByText(/Sale del inventario de la sede lo que entró con sus 2 líneas/)).toBeInTheDocument();
    await write(dialog.getByLabelText('Motivo'), 'La factura se registró dos veces');
    await userEvent.click(dialog.getByRole('button', { name: 'Anular compra' }));

    expect(await screen.findByText('Compra FE 1234 anulada')).toBeInTheDocument();
    expect(await api.calls(`POST /api/v1/purchases/${purchaseId}/void`)[0]?.json()).toEqual({
      reason: 'La factura se registró dos veces',
    });
    expect(await screen.findByText(/Motivo: La factura se registró dos veces/)).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Anular compra' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Registrar pago' })).not.toBeInTheDocument();
  });

  it('exige el motivo', async () => {
    const api = voidApi();
    renderApp(`/compras/${purchaseId}`);

    const dialog = await openVoid();
    await userEvent.click(dialog.getByRole('button', { name: 'Anular compra' }));

    expect(dialog.getByLabelText('Motivo')).toHaveAccessibleDescription(expect.stringContaining('Escriba el motivo'));
    expect(api.calls(`POST /api/v1/purchases/${purchaseId}/void`)).toHaveLength(0);
  });

  it('si ya se vendió parte del stock, nombra el producto y no la anula', async () => {
    voidApi(aPurchase(), {
      [`POST /api/v1/purchases/${purchaseId}/void`]: apiError(409, {
        code: 'insufficient_stock',
        message: 'Línea 2: Ya no hay…',
        details: { line: '1' },
      }),
    });
    renderApp(`/compras/${purchaseId}`);

    const dialog = await openVoid();
    await write(dialog.getByLabelText('Motivo'), 'Error');
    await userEvent.click(dialog.getByRole('button', { name: 'Anular compra' }));

    await waitFor(() =>
      expect(dialog.getByRole('alert')).toHaveTextContent(
        'No se puede anular: ya no está en la sede todo lo que entró de Camiseta básica (Azul · S). Se vendió o se movió.',
      ),
    );
  });

  it('una compra con abonos no se puede anular', async () => {
    voidApi(aPurchase({ balance_due: '18480.00' }));
    renderApp(`/compras/${purchaseId}`);

    expect(await screen.findByRole('button', { name: 'Registrar pago' })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Anular compra' })).not.toBeInTheDocument();
  });
});
