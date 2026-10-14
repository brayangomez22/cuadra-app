import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser, apiError, fakeApi, json, signedIn } from '@/test/fakeApi';
import { aClosedCashSession, aLocation, anOpenCashSession, ids, plain } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';

const user = () => userEvent.setup({ delay: null });

function cashApi(routes: Parameters<typeof fakeApi>[0] = {}) {
  return fakeApi({
    ...signedIn(aUser('cashier')),
    'GET /api/v1/locations': json(200, { items: [aLocation()] }),
    'GET /api/v1/sales/discount-limit': json(200, {
      unlimited: false,
      max: '0',
    }),
    'GET /api/v1/cash-sessions/current': json(200, { required: true }),
    ...routes,
  });
}

beforeEach(() => {
  sessionStorage.clear();
  localStorage.clear();
});

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

describe('CloseCashPage', () => {
  const closeKey = 'POST /api/v1/cash-sessions/01920000-0000-7000-8000-000000000401/close';

  function closeApi(routes: Parameters<typeof fakeApi>[0] = {}) {
    return cashApi({
      'GET /api/v1/cash-sessions/current': json(200, {
        required: true,
        session: anOpenCashSession(),
      }),
      'GET /api/v1/cash-sessions/01920000-0000-7000-8000-000000000401': json(200, aClosedCashSession()),
      [closeKey]: json(200, aClosedCashSession()),
      ...routes,
    });
  }

  it('suma lo contado por denominación mientras se escribe, sin mostrar el esperado', async () => {
    closeApi();
    renderApp('/caja/cerrar');

    await user().type(await screen.findByLabelText('Billetes de $ 100.000'), '1');
    await user().type(screen.getByLabelText('Billetes de $ 20.000'), '2');
    await user().type(screen.getByLabelText('Monedas de $ 500'), '3');

    const total = screen.getByRole('status', { name: 'Total contado' });
    await waitFor(() => expect(plain(total.textContent)).toContain('$ 141.500'));
    expect(screen.queryByText(/esperado/i)).not.toBeInTheDocument();
  });

  it('pide confirmar, cierra con el conteo y muestra el reporte', async () => {
    // Like the API: once closed, the user has no open session.
    let current: unknown = { required: true, session: anOpenCashSession() };
    const api = closeApi({
      'GET /api/v1/cash-sessions/current': () => json(200, current),
      [closeKey]: () => {
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
    const [request] = api.calls(closeKey);
    expect(await request!.json()).toEqual({
      counts: [
        { denomination: '100000', quantity: 1 },
        { denomination: '20000', quantity: 2 },
        { denomination: '5000', quantity: 1 },
      ],
      notes: 'Faltó un billete',
    });
  });

  it('rechaza una cantidad que no es un número entero', async () => {
    const api = closeApi();
    renderApp('/caja/cerrar');

    await user().type(await screen.findByLabelText('Billetes de $ 50.000'), '1,5');
    await user().click(screen.getByRole('button', { name: 'Cerrar caja' }));

    expect(await screen.findByText('Escriba cuántos hay, sin decimales')).toBeInTheDocument();
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    expect(api.calls(closeKey)).toHaveLength(0);
  });

  it('muestra el error de la API si la caja ya estaba cerrada', async () => {
    closeApi({
      [closeKey]: apiError(
        409,
        'cash_session_closed',
        'La caja ya está cerrada. Para seguir vendiendo, abra una nueva.',
      ),
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

describe('CashReportPage', () => {
  it('muestra el esperado, lo contado y el faltante del cierre', async () => {
    cashApi({
      'GET /api/v1/cash-sessions/01920000-0000-7000-8000-000000000401': json(200, aClosedCashSession()),
    });
    renderApp(`/caja/${ids.cashSession}`);

    const result = await screen.findByRole('region', {
      name: 'Resultado del arqueo',
    });
    expect(within(result).getByText('Faltante')).toBeInTheDocument();
    const value = (term: string) =>
      plain(within(result).getByText(term, { selector: 'dt' }).nextElementSibling?.textContent);
    expect(value('Esperado')).toBe('$ 148.000');
    expect(value('Contado')).toBe('$ 145.000');
    expect(value('Diferencia')).toBe('-$ 3.000');
  });

  it('imprime el reporte de cierre en el ancho del papel del equipo', async () => {
    localStorage.setItem('cuadra.pos.paper', '58');
    const print = vi.spyOn(window, 'print').mockImplementation(() => {});
    cashApi({
      'GET /api/v1/cash-sessions/01920000-0000-7000-8000-000000000401': json(200, aClosedCashSession()),
    });
    renderApp(`/caja/${ids.cashSession}`);

    await user().click(await screen.findByRole('button', { name: 'Imprimir cierre' }));

    expect(print).toHaveBeenCalledOnce();
    const report = document.querySelector('[data-print-root] [aria-label="Cierre de caja"]');
    expect(report).toHaveAttribute('data-paper', '58');
    expect(plain(report?.textContent)).toContain('Faltó un billete');
    expect(plain(report?.textContent)).toContain('$ 148.000');
  });

  it('una caja abierta todavía no tiene reporte', async () => {
    cashApi({
      'GET /api/v1/cash-sessions/01920000-0000-7000-8000-000000000401': json(200, anOpenCashSession()),
    });
    renderApp(`/caja/${ids.cashSession}`);

    expect(await screen.findByText('La caja sigue abierta')).toBeInTheDocument();
  });
});
