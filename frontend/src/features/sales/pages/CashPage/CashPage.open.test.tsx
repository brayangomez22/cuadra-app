import { screen, waitFor } from '@testing-library/react';
import { json } from '@/test/fakeApi';
import { aLocation, ids } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { cashApi } from '../../testing/cashApi';
import { anOpenCashSession } from '../../testing/fixtures';
import { clearPosStorage, user } from '../../testing/posSteps';

beforeEach(clearPosStorage);

describe('CashPage', () => {
  it('sin caja abierta, abre la caja con su base y lleva a vender', async () => {
    let current: unknown = { required: true };
    const api = cashApi({
      'GET /api/v1/cash-sessions/current': () => json(200, current),
      'POST /api/v1/cash-sessions': () => {
        current = {
          required: true,
          session: anOpenCashSession({ movements: [] }),
        };
        return json(201, anOpenCashSession({ movements: [] }));
      },
    });
    const { router } = renderApp('/caja');

    await user().type(await screen.findByLabelText('Base inicial'), '150.000');
    await user().click(screen.getByRole('button', { name: 'Abrir caja' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/vender'));
    const [request] = api.calls('POST /api/v1/cash-sessions');
    expect(await request!.json()).toEqual({
      location_id: ids.main,
      opening_float: '150000',
    });
  });

  it('con varias sedes pide elegir dónde abrir la caja', async () => {
    const api = cashApi({
      'GET /api/v1/locations': json(200, {
        items: [aLocation(), aLocation({ id: ids.north, name: 'Sede norte' })],
      }),
    });
    renderApp('/caja');

    await user().type(await screen.findByLabelText('Base inicial'), '0');
    await user().click(screen.getByRole('button', { name: 'Abrir caja' }));

    expect(await screen.findByText('Elija la sede de la caja')).toBeInTheDocument();
    expect(api.calls('POST /api/v1/cash-sessions')).toHaveLength(0);
  });

  it('rechaza una base inválida sin llamar a la API', async () => {
    const api = cashApi();
    renderApp('/caja');

    await user().type(await screen.findByLabelText('Base inicial'), 'cien');
    await user().click(screen.getByRole('button', { name: 'Abrir caja' }));

    expect(await screen.findByText(/Escriba un valor/)).toBeInTheDocument();
    expect(api.calls('POST /api/v1/cash-sessions')).toHaveLength(0);
  });
});
