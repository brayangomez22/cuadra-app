import { screen, waitFor, within } from '@testing-library/react';
import { apiError, json } from '@/test/fakeApi';
import { ids, plain } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { CLOSE_KEY, cashApi, closeApi } from '../../testing/cashApi';
import { aClosedCashSession, anOpenCashSession } from '../../testing/fixtures';
import { clearPosStorage, user } from '../../testing/posSteps';

beforeEach(clearPosStorage);

describe('CloseCashPage', () => {
  it('pide confirmar, cierra con el conteo y muestra el reporte', async () => {
    // Like the API: once closed, the user has no open session.
    let current: unknown = { required: true, session: anOpenCashSession() };
    const api = closeApi({
      'GET /api/v1/cash-sessions/current': () => json(200, current),
      [CLOSE_KEY]: () => {
        current = { required: true };
        return json(200, aClosedCashSession());
      },
    });
    const { router } = renderApp('/caja/cerrar');

    await user().type(await screen.findByLabelText('Billetes de $ 100.000'), '1');
    await user().type(screen.getByLabelText('Billetes de $ 20.000'), '2');
    await user().type(screen.getByLabelText('Billetes de $ 5.000'), '1');
    await user().type(screen.getByLabelText('Notas (opcional)'), 'Faltó un billete');
    await user().click(screen.getByRole('button', { name: 'Cerrar caja' }));

    const confirm = await screen.findByRole('dialog', {
      name: '¿Cerrar la caja?',
    });
    expect(plain(confirm.textContent)).toContain('$ 145.000');
    expect(within(confirm).getByRole('button', { name: 'Seguir contando' })).toHaveFocus();
    await user().click(within(confirm).getByRole('button', { name: 'Cerrar caja' }));

    expect(await screen.findByRole('heading', { name: 'Cierre de caja' })).toBeInTheDocument();
    expect(router.state.location.pathname).toBe(`/caja/${ids.cashSession}`);
    const [request] = api.calls(CLOSE_KEY);
    expect(await request!.json()).toEqual({
      counts: [
        { denomination: '100000', quantity: 1 },
        { denomination: '20000', quantity: 2 },
        { denomination: '5000', quantity: 1 },
      ],
      notes: 'Faltó un billete',
    });
  });

  it('muestra el error de la API si la caja ya estaba cerrada', async () => {
    closeApi({
      [CLOSE_KEY]: apiError(409, {
        code: 'cash_session_closed',
        message: 'La caja ya está cerrada. Para seguir vendiendo, abra una nueva.',
      }),
    });
    renderApp('/caja/cerrar');

    await user().click(await screen.findByRole('button', { name: 'Cerrar caja' }));
    await user().click(
      within(await screen.findByRole('dialog')).getByRole('button', {
        name: 'Cerrar caja',
      }),
    );

    expect(await screen.findByText(/La caja ya está cerrada/)).toBeInTheDocument();
  });

  it('sin caja abierta lleva a abrirla', async () => {
    cashApi();
    const { router } = renderApp('/caja/cerrar');

    await waitFor(() => expect(router.state.location.pathname).toBe('/caja'));
  });
});
