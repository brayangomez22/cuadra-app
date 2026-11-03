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

// The sale's drawer closed yesterday; the user has today's open.
const myDrawer = '01920000-0000-7000-8000-000000000d77';
const MOVE_KEY = `POST /api/v1/cash-sessions/${myDrawer}/movements` as const;
const movement = {
  id: ids.cashSession,
  kind: 'out',
  amount: '11900.00',
  reason: 'x',
  user_id: ids.cashSession,
  created_at: '2026-10-14T15:00:00Z',
} as const;

function voidApi(routes: Parameters<typeof salesApi>[0] = {}) {
  return salesApi({
    [SALE_KEY]: json(200, aSale({ cash_session_id: ids.cashSession })),
    [CASH_KEY]: json(200, aClosedCashSession()),
    'GET /api/v1/cash-sessions/current': json(200, { required: true, session: anOpenCashSession({ id: myDrawer }) }),
    [VOID_KEY]: json(200, aVoidedSale()),
    [MOVE_KEY]: json(201, movement),
    ...routes,
  });
}

async function voidWithReason() {
  await user().click(await screen.findByRole('button', { name: 'Anular venta' }));
  const dialog = within(await screen.findByRole('dialog', { name: 'Anular la venta N.º 15' }));
  await user().type(dialog.getByRole('textbox', { name: 'Motivo' }), 'Devolvió todo');
  return dialog;
}

const confirm = () => within(screen.getByRole('dialog')).getByRole('button', { name: 'Anular venta' });

describe('SaleDetailPage: el efectivo devuelto sale de la caja abierta', () => {
  it('con la caja de la venta cerrada saca el efectivo de la caja propia como egreso', async () => {
    const api = voidApi();
    renderApp(`/ventas/${saleId}`);

    const dialog = await voidWithReason();
    expect(await dialog.findByRole('checkbox', { name: /Sacar \$\s11\.900 de mi caja abierta/ })).toBeChecked();
    await user().click(confirm());

    await waitFor(() => expect(api.calls(MOVE_KEY)).toHaveLength(1));
    expect(await api.calls(MOVE_KEY)[0]!.json()).toEqual({
      kind: 'out',
      amount: '11900.00',
      reason: 'Devolución por anular la venta N.º 15',
    });
    expect(await screen.findByText(/Salieron \$\s11\.900 de su caja/)).toBeInTheDocument();
  });

  it('desmarcada no toca la caja', async () => {
    const api = voidApi();
    renderApp(`/ventas/${saleId}`);

    const dialog = await voidWithReason();
    await user().click(await dialog.findByRole('checkbox', { name: /Sacar/ }));
    await user().click(confirm());

    await waitFor(() => expect(api.calls(VOID_KEY)).toHaveLength(1));
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    expect(api.calls(MOVE_KEY)).toHaveLength(0);
  });

  it('sin caja propia abierta solo avisa que registre el egreso', async () => {
    voidApi({ 'GET /api/v1/cash-sessions/current': json(200, { required: true }) });
    renderApp(`/ventas/${saleId}`);

    const dialog = await voidWithReason();
    expect(await dialog.findByText(/regístrelo como egreso de la caja actual/)).toBeInTheDocument();
    expect(dialog.queryByRole('checkbox')).not.toBeInTheDocument();
  });

  it('si el egreso falla dice que la venta sí quedó anulada y que lo registre en la caja', async () => {
    voidApi({
      [MOVE_KEY]: json(409, { error: { code: 'cash_session_closed', message: 'La caja ya está cerrada' } } as never),
    });
    renderApp(`/ventas/${saleId}`);

    await voidWithReason();
    await user().click(confirm());

    expect(await screen.findByText(/no pudimos registrar el egreso/)).toBeInTheDocument();
  });
});
