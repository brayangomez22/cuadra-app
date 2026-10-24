import { screen, waitFor, within } from '@testing-library/react';
import { json } from '@/test/fakeApi';
import { plain } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { cashApi } from '../../testing/cashApi';
import { anOpenCashSession } from '../../testing/fixtures';
import { clearPosStorage, user } from '../../testing/posSteps';

beforeEach(clearPosStorage);

describe('CashPage', () => {
  it('muestra la caja abierta con su base y sus movimientos, sin el esperado', async () => {
    cashApi({
      'GET /api/v1/cash-sessions/current': json(200, {
        required: true,
        session: anOpenCashSession(),
      }),
    });
    renderApp('/caja');

    const movements = await screen.findByRole('table', {
      name: 'Ingresos y egresos',
    });
    expect(within(movements).getByText('Pago del domicilio')).toBeInTheDocument();
    expect(plain(within(movements).getByText(/12\.000/).textContent)).toBe('-$ 12.000');
    expect(screen.getByText('Bodega principal')).toBeInTheDocument();
    expect(plain(screen.getByText(/100\.000/).textContent)).toBe('$ 100.000');
    expect(screen.queryByText(/esperado/i)).not.toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Cerrar caja' })).toHaveAttribute('href', '/caja/cerrar');
  });

  it('registra un egreso con su motivo', async () => {
    const api = cashApi({
      'GET /api/v1/cash-sessions/current': json(200, {
        required: true,
        session: anOpenCashSession({ movements: [] }),
      }),
      'POST /api/v1/cash-sessions/01920000-0000-7000-8000-000000000401/movements': json(
        201,
        anOpenCashSession().movements[0],
      ),
    });
    renderApp('/caja');

    await user().click(await screen.findByRole('button', { name: 'Registrar egreso' }));
    const dialog = await screen.findByRole('dialog', {
      name: 'Registrar egreso',
    });
    await user().type(within(dialog).getByLabelText('Valor'), '12.000');
    await user().type(within(dialog).getByLabelText('Motivo'), 'Pago del domicilio');
    await user().click(within(dialog).getByRole('button', { name: 'Registrar egreso' }));

    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    const [request] = api.calls('POST /api/v1/cash-sessions/01920000-0000-7000-8000-000000000401/movements');
    expect(await request!.json()).toEqual({
      kind: 'out',
      amount: '12000',
      reason: 'Pago del domicilio',
    });
  });

  it('pide el motivo del movimiento', async () => {
    const api = cashApi({
      'GET /api/v1/cash-sessions/current': json(200, {
        required: true,
        session: anOpenCashSession(),
      }),
    });
    renderApp('/caja');

    await user().click(await screen.findByRole('button', { name: 'Registrar ingreso' }));
    const dialog = await screen.findByRole('dialog', {
      name: 'Registrar ingreso',
    });
    await user().type(within(dialog).getByLabelText('Valor'), '5000');
    await user().click(within(dialog).getByRole('button', { name: 'Registrar ingreso' }));

    expect(await within(dialog).findByText('Escriba el motivo')).toBeInTheDocument();
    expect(api.calls('POST /api/v1/cash-sessions/01920000-0000-7000-8000-000000000401/movements')).toHaveLength(0);
  });
});
