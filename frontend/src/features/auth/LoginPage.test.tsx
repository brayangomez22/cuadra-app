import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser, anAuthSession, apiError, fakeApi, json, signedOut } from '@/test/fakeApi';
import { renderApp } from '@/test/renderApp';

async function fillLogin(email = 'ana@eltornillo.co', password = 'clave-segura') {
  await userEvent.type(await screen.findByLabelText('Correo'), email);
  await userEvent.type(screen.getByLabelText('Contraseña'), password);
  await userEvent.click(screen.getByRole('button', { name: 'Ingresar' }));
}

describe('LoginPage', () => {
  it('muestra errores de validación al enviar el formulario vacío', async () => {
    const api = fakeApi(signedOut());
    renderApp('/ingresar');

    await userEvent.click(await screen.findByRole('button', { name: 'Ingresar' }));

    expect(screen.getByLabelText('Correo')).toHaveAccessibleDescription('Escriba su correo');
    expect(screen.getByLabelText('Correo')).toHaveAttribute('aria-invalid', 'true');
    expect(screen.getByLabelText('Contraseña')).toHaveAccessibleDescription('Escriba su contraseña');
    expect(api.calls('POST /api/v1/auth/login')).toHaveLength(0);
  });

  it('rechaza un correo con formato inválido sin llamar a la API', async () => {
    const api = fakeApi(signedOut());
    renderApp('/ingresar');

    await fillLogin('ana-eltornillo.co');

    expect(screen.getByLabelText('Correo')).toHaveAccessibleDescription(
      'Escriba un correo válido, como ana@minegocio.co',
    );
    expect(api.calls('POST /api/v1/auth/login')).toHaveLength(0);
  });

  it('inicia sesión y vuelve a la ruta que se había pedido', async () => {
    const api = fakeApi({ ...signedOut(), 'POST /api/v1/auth/login': json(200, anAuthSession()) });
    const { router } = renderApp('/inventario');
    await waitFor(() => expect(router.state.location.pathname).toBe('/ingresar'));

    await fillLogin();

    await waitFor(() => expect(router.state.location.pathname).toBe('/inventario'));
    expect(await screen.findByRole('heading', { level: 1, name: 'Inventario' })).toBeInTheDocument();
    expect(await api.calls('POST /api/v1/auth/login')[0]?.json()).toEqual({
      email: 'ana@eltornillo.co',
      password: 'clave-segura',
    });
  });

  it('muestra el mensaje de credenciales incorrectas ante un 401', async () => {
    fakeApi({
      ...signedOut(),
      'POST /api/v1/auth/login': apiError(401, 'invalid_credentials', 'Correo o contraseña incorrectos.'),
    });
    renderApp('/ingresar');

    await fillLogin();

    expect(await screen.findByRole('alert')).toHaveTextContent('Correo o contraseña incorrectos.');
  });

  it('pide elegir la empresa ante un 409 y repite el login con el tenant elegido', async () => {
    const tenants = [
      { id: '01920000-0000-7000-8000-0000000000aa', name: 'Ferretería El Tornillo' },
      { id: '01920000-0000-7000-8000-0000000000bb', name: 'Depósito La Tuerca' },
    ];
    let attempt = 0;
    const api = fakeApi({
      ...signedOut(),
      'POST /api/v1/auth/login': () =>
        ++attempt === 1
          ? json(409, {
              error: { code: 'tenant_selection_required', message: 'Elija la empresa' },
              tenants,
            })
          : json(200, anAuthSession(aUser())),
    });
    const { router } = renderApp('/ingresar');

    await fillLogin();
    await userEvent.click(await screen.findByRole('button', { name: 'Depósito La Tuerca' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/'));
    const [, second] = api.calls('POST /api/v1/auth/login');
    expect(await second?.json()).toEqual({
      email: 'ana@eltornillo.co',
      password: 'clave-segura',
      tenant_id: '01920000-0000-7000-8000-0000000000bb',
    });
  });

  it('muestra el aviso de demasiados intentos ante un 429', async () => {
    fakeApi({
      ...signedOut(),
      'POST /api/v1/auth/login': apiError(429, 'too_many_requests', 'Demasiados intentos. Espere un minuto.'),
    });
    renderApp('/ingresar');

    await fillLogin();

    expect(await screen.findByRole('alert')).toHaveTextContent('Demasiados intentos. Espere un minuto.');
  });
});
