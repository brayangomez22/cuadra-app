import { screen, waitFor, within } from '@testing-library/react';
import { json } from '@/test/fakeApi';
import { ids } from '@/test/fixtures';
import { preloadPages } from '@/test/preloadPages';
import { renderApp } from '@/test/renderApp';
import { aClosedCashSession, anOpenCashSession, aSale } from '../../testing/fixtures';
import { clearPosStorage, user } from '../../testing/posSteps';
import { aVoidedSale, CASH_KEY, SALE_KEY, saleId, salesApi, VOID_KEY } from '../../testing/salesApi';

beforeAll(preloadPages('/ventas/x'));
beforeEach(clearPosStorage);

const inDrawer = aSale({ cash_session_id: ids.cashSession });

async function openVoid() {
  await user().click(await screen.findByRole('button', { name: 'Anular venta' }));
  return within(await screen.findByRole('dialog', { name: 'Anular la venta N.º 15' }));
}

const confirm = () => within(screen.getByRole('dialog')).getByRole('button', { name: 'Anular venta' });

describe('SaleDetailPage: anular', () => {
  it('exige el motivo antes de anular', async () => {
    const api = salesApi({ [VOID_KEY]: json(200, aVoidedSale()) });
    renderApp(`/ventas/${saleId}`);

    const dialog = await openVoid();
    await user().click(confirm());

    expect(await dialog.findByText('Escriba el motivo')).toBeInTheDocument();
    expect(api.calls(VOID_KEY)).toHaveLength(0);
  });

  it('con la caja abierta avisa que devuelva el efectivo', async () => {
    salesApi({ [SALE_KEY]: json(200, inDrawer), [CASH_KEY]: json(200, anOpenCashSession()) });
    renderApp(`/ventas/${saleId}`);

    const dialog = await openVoid();
    expect(await dialog.findByText(/La caja deja de contar esta venta/)).toBeInTheDocument();
  });

  it('con la caja cerrada avisa que registre el egreso', async () => {
    salesApi({ [SALE_KEY]: json(200, inDrawer), [CASH_KEY]: json(200, aClosedCashSession()) });
    renderApp(`/ventas/${saleId}`);

    const dialog = await openVoid();
    expect(await dialog.findByText(/regístrelo como egreso de la caja actual/)).toBeInTheDocument();
  });

  it('sin saber el estado de la caja no deja anular', async () => {
    salesApi({
      [SALE_KEY]: json(200, inDrawer),
      [CASH_KEY]: json(500, { error: { code: 'internal', message: 'Falló' } } as never),
    });
    renderApp(`/ventas/${saleId}`);

    const dialog = await openVoid();
    expect(await dialog.findByText(/No pudimos revisar la caja/)).toBeInTheDocument();
    expect(confirm()).toBeDisabled();
  });

  it('anula con el motivo, cierra y la tirilla queda ANULADA', async () => {
    const api = salesApi({ [VOID_KEY]: json(200, aVoidedSale()) });
    renderApp(`/ventas/${saleId}`);

    const dialog = await openVoid();
    await user().type(dialog.getByRole('textbox', { name: 'Motivo' }), '  El cliente desistió ');
    await user().click(confirm());

    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    expect(await api.calls(VOID_KEY)[0]!.json()).toEqual({ reason: 'El cliente desistió' });
    const receipt = within(await screen.findByRole('article', { name: 'Tirilla de la venta N.º 15' }));
    expect(await receipt.findByText('ANULADA')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Anular venta' })).not.toBeInTheDocument();
  });

  it('muestra el rechazo de una venta a crédito con abonos y no la da por anulada', async () => {
    const message = 'El cliente ya abonó a esta venta.';
    salesApi({ [VOID_KEY]: json(409, { error: { code: 'credit_sale_has_payments', message } } as never) });
    renderApp(`/ventas/${saleId}`);

    const dialog = await openVoid();
    await user().type(dialog.getByRole('textbox', { name: 'Motivo' }), 'Error del cajero');
    await user().click(confirm());

    expect(await dialog.findByText(message)).toBeInTheDocument();
    expect(screen.queryByText('ANULADA')).not.toBeInTheDocument();
  });

  it('marca en el campo el motivo que la API rechaza', async () => {
    const message = 'El motivo es muy largo.';
    salesApi({ [VOID_KEY]: json(422, { error: { code: 'invalid_void_reason', message } } as never) });
    renderApp(`/ventas/${saleId}`);

    const dialog = await openVoid();
    const reason = dialog.getByRole('textbox', { name: 'Motivo' });
    await user().type(reason, 'x');
    await user().click(confirm());

    await waitFor(() => expect(reason).toHaveAttribute('aria-invalid', 'true'));
    expect(dialog.getByText(message)).toBeInTheDocument();
  });
});
