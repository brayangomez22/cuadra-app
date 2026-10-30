import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser, anAuthSession, apiError, fakeApi, json, signedOut } from '@/test/fakeApi';
import { renderApp } from '@/test/renderApp';
import { demoApi } from '../../testing/demoApi';

const demoUser = {
  ...aUser('cashier'),
  tenant_name: 'Almacén de Ropa Demo',
  business_template: 'ropa_calzado',
  demo: true,
};

describe('DemoPage', () => {
  it('entra a la demo del tipo de negocio con el rol elegido; el cajero empieza en el POS', async () => {
    const api = demoApi({ 'POST /api/v1/auth/demo': json(200, anAuthSession(demoUser)) });
    const { router } = renderApp('/demo');

    await userEvent.click(await screen.findByRole('radio', { name: 'Ropa y calzado' }));
    await userEvent.click(screen.getByRole('radio', { name: 'Cajero' }));
    await userEvent.click(screen.getByRole('button', { name: 'Entrar a la demo' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/vender'));
    expect(await api.calls('POST /api/v1/auth/demo')[0]?.json()).toEqual({
      business_template: 'ropa_calzado',
      role: 'cashier',
    });
    expect(await screen.findByRole('complementary', { name: 'Modo demostración' })).toBeInTheDocument();
  });

  it('propone el primer tipo de negocio y el rol de dueño', async () => {
    const api = demoApi({ 'POST /api/v1/auth/demo': json(200, anAuthSession({ ...aUser('owner'), demo: true })) });
    renderApp('/demo');

    expect(await screen.findByRole('radio', { name: 'Ferretería' })).toBeChecked();
    expect(screen.getByRole('radio', { name: 'Dueño' })).toBeChecked();
    await userEvent.click(screen.getByRole('button', { name: 'Entrar a la demo' }));

    await waitFor(() => expect(api.calls('POST /api/v1/auth/demo')).toHaveLength(1));
    expect(await api.calls('POST /api/v1/auth/demo')[0]?.json()).toEqual({
      business_template: 'ferreteria',
      role: 'owner',
    });
  });

  it('el dueño empieza en el inicio', async () => {
    demoApi({ 'POST /api/v1/auth/demo': json(200, anAuthSession({ ...aUser('owner'), demo: true })) });
    const { router } = renderApp('/demo');

    await userEvent.click(await screen.findByRole('button', { name: 'Entrar a la demo' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/'));
  });

  it('muestra el error de la API si la demo dejó de estar disponible', async () => {
    demoApi({
      'POST /api/v1/auth/demo': apiError(404, {
        code: 'demo_unavailable',
        message: 'La demostración no está disponible.',
      }),
    });
    renderApp('/demo');

    await userEvent.click(await screen.findByRole('button', { name: 'Entrar a la demo' }));

    expect(await screen.findByText('La demostración no está disponible.')).toBeInTheDocument();
  });

  it('con la demo apagada lo dice y ofrece ingresar', async () => {
    fakeApi({ ...signedOut(), 'GET /api/v1/auth/demo': apiError(404, { code: 'demo_unavailable', message: 'No' }) });
    renderApp('/demo');

    expect(await screen.findByRole('heading', { name: 'La demostración no está disponible' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Ingresar' })).toHaveAttribute('href', '/ingresar');
    expect(screen.queryByRole('button', { name: 'Entrar a la demo' })).not.toBeInTheDocument();
  });
});
