import { screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser, apiError, json } from '@/test/fakeApi';
import { aCredit, customerIds, plain } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { builderAccount, receivablesApi } from '../../testing/receivablesApi';

const user = () => userEvent.setup({ delay: null });

// The figure of the credit's summary by its term ("Cupo", "Debe").
const figure = (term: string) => {
  const summary = within(screen.getByRole('region', { name: 'Crédito' }));
  return plain(summary.getByText(term, { selector: 'dt' }).nextElementSibling?.textContent);
};

beforeEach(() => {
  vi.useFakeTimers({ toFake: ['Date'] });
  vi.setSystemTime(new Date('2026-10-28T15:00:00Z'));
});
afterEach(() => vi.useRealTimers());

describe('CustomerAccountPage', () => {
  it('muestra el cupo, el plazo, lo que debe, lo disponible y las cuentas abiertas', async () => {
    receivablesApi();
    renderApp(builderAccount);

    expect(await screen.findByRole('heading', { name: 'Carlos Pérez', level: 1 })).toBeInTheDocument();
    expect(await screen.findByText('Cupo', { selector: 'dt' })).toBeInTheDocument();
    expect(figure('Cupo')).toBe('$ 1.500.000');
    expect(figure('Plazo')).toBe('30 días');
    expect(figure('Debe')).toBe('$ 350.000');
    expect(figure('Disponible')).toBe('$ 1.150.000');
    const open = within(screen.getByRole('table', { name: 'Cuentas por cobrar abiertas' }));
    const overdue = within(open.getByRole('row', { name: /Vencida hace 8 días/ }));
    expect(plain(overdue.getAllByRole('cell')[2]?.textContent)).toBe('$ 150.000');
    expect(open.getByRole('row', { name: /Vence el 25\/11\/2026/ })).toBeInTheDocument();
  });

  it('guarda el cupo y el plazo', async () => {
    const api = receivablesApi({
      routes: {
        'PUT /api/v1/customers/{customerId}/credit': json(200, aCredit({ credit_limit: '2000000.00', term_days: 45 })),
      },
    });
    renderApp(builderAccount);

    const limit = await screen.findByLabelText('Cupo de crédito');
    expect(limit).toHaveValue('1500000');
    await user().clear(limit);
    await user().type(limit, '2.000.000');
    await user().clear(screen.getByLabelText('Plazo en días'));
    await user().type(screen.getByLabelText('Plazo en días'), '45');
    await user().click(screen.getByRole('button', { name: 'Guardar cupo' }));

    expect(await screen.findByText('Cupo guardado')).toBeInTheDocument();
    const [put] = api.calls(`PUT /api/v1/customers/${customerIds.builder}/credit`);
    expect(await put!.json()).toEqual({ credit_limit: '2000000', term_days: 45 });
  });

  it('lleva el error del plazo de la API a su campo', async () => {
    receivablesApi({
      routes: {
        'PUT /api/v1/customers/{customerId}/credit': apiError(422, {
          code: 'invalid_credit_term',
          message: 'El plazo no es válido',
        }),
      },
    });
    renderApp(builderAccount);

    await user().click(await screen.findByRole('button', { name: 'Guardar cupo' }));

    expect(await screen.findByLabelText('Plazo en días')).toHaveAccessibleDescription(/El plazo no es válido/);
  });

  it('el cajero ve la cuenta y recibe abonos, pero no cambia el cupo', async () => {
    receivablesApi({ user: aUser('cashier') });
    renderApp(builderAccount);

    expect(await screen.findByRole('button', { name: 'Registrar abono' })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Guardar cupo' })).not.toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Editar cliente' })).toBeInTheDocument();
  });

  it('un cliente sin cupo ni deuda no puede abonar y lo dice', async () => {
    receivablesApi();
    renderApp(`/clientes/${customerIds.boutique}/cuenta`);

    expect(await screen.findByRole('heading', { name: 'Confecciones Sol SAS', level: 1 })).toBeInTheDocument();
    expect(await screen.findByText('No debe nada')).toBeInTheDocument();
    expect(screen.getByText(/Con cupo 0 no puede comprar a crédito/)).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Registrar abono' })).not.toBeInTheDocument();
  });
});
