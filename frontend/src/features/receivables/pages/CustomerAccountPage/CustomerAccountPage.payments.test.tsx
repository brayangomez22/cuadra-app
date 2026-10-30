import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { apiError, json } from '@/test/fakeApi';
import { customerIds, page, plain, query, statement } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { builderAccount, receivablesApi } from '../../testing/receivablesApi';

const user = () => userEvent.setup({ delay: null });

const payment = {
  id: '01920000-0000-7000-8000-000000000721',
  customer_id: customerIds.builder,
  method: 'nequi' as const,
  amount: '100000.00',
  created_at: '2026-10-28T15:00:00Z',
  allocations: [{ receivable_id: '01920000-0000-7000-8000-000000000711', amount: '100000.00' }],
  balance: '250000.00',
};

// openPayment waits for the account and opens the payment's dialog.
async function openPayment() {
  await user().click(await screen.findByRole('button', { name: 'Registrar abono' }));
  return within(await screen.findByRole('dialog', { name: 'Registrar abono' }));
}

describe('CustomerAccountPage: abonos', () => {
  it('registra un abono y dice cuánto queda debiendo', async () => {
    const api = receivablesApi({ routes: { 'POST /api/v1/customers/{customerId}/payments': json(201, payment) } });
    renderApp(builderAccount);

    const dialog = await openPayment();
    expect(plain(dialog.getByText(/^Debe/).textContent)).toBe(
      'Debe $ 350.000. El abono se aplica a la venta que vence primero.',
    );
    await user().click(dialog.getByRole('combobox', { name: 'Medio de pago' }));
    await user().click(await screen.findByRole('option', { name: 'Nequi' }));
    await user().type(dialog.getByLabelText('Valor del abono'), '100.000');
    await user().click(dialog.getByRole('button', { name: 'Registrar abono' }));

    expect(await screen.findByText('Abono registrado')).toBeInTheDocument();
    expect(plain(screen.getByText(/^Queda debiendo/).textContent)).toBe('Queda debiendo $ 250.000');
    const [post] = api.calls(`POST /api/v1/customers/${customerIds.builder}/payments`);
    expect(await post!.json()).toEqual({ method: 'nequi', amount: '100000' });
    await waitFor(() => expect(api.calls(`GET /api/v1/customers/${customerIds.builder}/credit`)).toHaveLength(2));
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('no deja abonar más de lo que debe', async () => {
    const api = receivablesApi();
    renderApp(builderAccount);

    const dialog = await openPayment();
    await user().type(dialog.getByLabelText('Valor del abono'), '400000');
    await user().click(dialog.getByRole('button', { name: 'Registrar abono' }));

    expect(plain((await dialog.findByText(/no puede pasar/)).textContent)).toBe(
      'El abono no puede pasar de lo que debe ($ 350.000)',
    );
    expect(api.calls(`POST /api/v1/customers/${customerIds.builder}/payments`)).toHaveLength(0);
  });

  it('un abono en efectivo sin caja abierta muestra el error de la API', async () => {
    receivablesApi({
      routes: {
        'POST /api/v1/customers/{customerId}/payments': apiError(409, {
          code: 'cash_session_required',
          message: 'Abra la caja antes de recibir efectivo',
        }),
      },
    });
    renderApp(builderAccount);

    const dialog = await openPayment();
    await user().type(dialog.getByLabelText('Valor del abono'), '50000');
    await user().click(dialog.getByRole('button', { name: 'Registrar abono' }));

    expect(await dialog.findByRole('alert')).toHaveTextContent('Abra la caja antes de recibir efectivo');
  });

  it('muestra el estado de cuenta con el saldo de cada movimiento, por páginas', async () => {
    const api = receivablesApi({
      routes: {
        'GET /api/v1/customers/{customerId}/credit/movements': json(200, {
          ...page(statement, 23),
          balance: '350000.00',
        }),
      },
    });
    renderApp(builderAccount);

    const table = within(await screen.findByRole('table', { name: 'Estado de cuenta' }));
    const paid = within(await table.findByRole('row', { name: /Abono · Efectivo/ }));
    expect(paid.getAllByRole('cell').map((c) => plain(c.textContent))).toEqual([
      '05/10/2026, 10:00 a. m.',
      'Abono · Efectivo05/10/2026, 10:00 a. m. · -$ 50.000',
      '-$ 50.000',
      '$ 150.000',
    ]);
    expect(table.getByRole('row', { name: /Venta a crédito.*Vence el 25\/11\/2026/ })).toBeInTheDocument();

    await user().click(screen.getByRole('button', { name: 'Siguiente' }));

    await waitFor(() => {
      const sent = query(api.calls(`GET /api/v1/customers/${customerIds.builder}/credit/movements`).at(-1)!);
      expect(sent.get('offset')).toBe('20');
    });
  });
});
