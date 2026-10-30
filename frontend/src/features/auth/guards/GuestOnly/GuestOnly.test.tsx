import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser, anAuthSession, fakeApi, json, signedIn, signedOut } from '@/test/fakeApi';
import { renderApp } from '@/test/renderApp';
import { demoApi } from '../../testing/demoApi';

describe('GuestOnly', () => {
  it('con una sesión abierta, /demo lo dice en vez de llevar a la app sin avisar', async () => {
    fakeApi(signedIn());
    const { router } = renderApp('/demo');

    expect(await screen.findByRole('heading', { name: 'Ya tiene una sesión abierta' })).toBeInTheDocument();
    expect(screen.getByText(/Ana Gómez · Dueño en Ferretería El Tornillo/)).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Volver a la app' })).toHaveAttribute('href', '/');
    expect(router.state.location.pathname).toBe('/demo');
  });

  it('cerrar la sesión desde /demo muestra las demos para elegir otra', async () => {
    const api = demoApi({
      ...signedIn({ ...aUser('cashier'), demo: true }),
      'POST /api/v1/auth/logout': new Response(null, { status: 204 }),
    });
    renderApp('/demo');

    await userEvent.click(await screen.findByRole('button', { name: 'Cerrar sesión y elegir otra demo' }));

    expect(await screen.findByRole('button', { name: 'Entrar a la demo' })).toBeInTheDocument();
    expect(api.calls('POST /api/v1/auth/logout')).toHaveLength(1);
  });

  it('en /ingresar ofrece entrar con otra cuenta', async () => {
    fakeApi({ ...signedIn(), 'POST /api/v1/auth/logout': new Response(null, { status: 204 }) });
    renderApp('/ingresar');

    await userEvent.click(await screen.findByRole('button', { name: 'Cerrar sesión y entrar con otra cuenta' }));

    expect(await screen.findByRole('button', { name: 'Ingresar' })).toBeInTheDocument();
  });

  it('quien ingresa en la página sí va a la app, sin el aviso', async () => {
    fakeApi({ ...signedOut(), 'POST /api/v1/auth/login': json(200, anAuthSession()) });
    const { router } = renderApp('/ingresar');

    await userEvent.type(await screen.findByLabelText(/Correo/), 'ana@eltornillo.co');
    await userEvent.type(screen.getByLabelText(/Contraseña/), 'una-clave-segura');
    await userEvent.click(screen.getByRole('button', { name: 'Ingresar' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/'));
    expect(screen.queryByRole('heading', { name: 'Ya tiene una sesión abierta' })).not.toBeInTheDocument();
  });
});
