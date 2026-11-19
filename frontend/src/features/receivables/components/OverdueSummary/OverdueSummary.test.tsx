import { screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser, apiError, fakeApi, json, signedIn } from '@/test/fakeApi';
import { buckets, plain, query } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { aging } from '../../testing/receivablesApi';

const agingKey = 'GET /api/v1/receivables/aging';

const card = async () => within(await screen.findByRole('region', { name: 'Cartera vencida' }));

describe('OverdueSummary en el inicio', () => {
  it('muestra lo vencido de todos los clientes y lo que pasa de 90 días, como la cartera', async () => {
    const api = fakeApi({ ...signedIn(aUser('admin')), [agingKey]: json(200, aging) });
    renderApp('/');

    const overdue = await card();
    expect(await overdue.findByText('$ 230.000')).toBeInTheDocument();
    expect(plain(overdue.getByText(/más de 90 días/).textContent)).toBe('$ 80.000 a más de 90 días');
    expect(query(api.calls(agingKey)[0]!).get('limit')).toBe('1');

    await userEvent.click(overdue.getByRole('link', { name: 'Ver la cartera' }));
    expect(plain((await screen.findByText('Vencida', { selector: 'dt' })).nextElementSibling?.textContent)).toBe(
      '$ 230.000',
    );
  });

  it('sin cartera vencida lo dice sin alarma', async () => {
    fakeApi({ ...signedIn(), [agingKey]: json(200, { ...aging, totals: buckets({ current: '200000.00' }) }) });
    renderApp('/');

    expect(await (await card()).findByText('Nadie tiene cuentas vencidas.')).toBeInTheDocument();
  });

  it('si falla muestra el error dentro de la tarjeta', async () => {
    fakeApi({ ...signedIn(), [agingKey]: apiError(500, { code: 'internal', message: 'Falló' }) });
    renderApp('/');

    expect(await (await card()).findByText('No pudimos cargar la cartera')).toBeInTheDocument();
  });
});
