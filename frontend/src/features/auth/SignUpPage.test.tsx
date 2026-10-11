import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { anAuthSession, apiError, fakeApi, json, signedOut } from '@/test/fakeApi';
import { renderApp } from '@/test/renderApp';

async function fillSignUp({ password = 'clave-segura' } = {}) {
  await userEvent.type(await screen.findByLabelText('Nombre de la empresa'), 'Ferretería El Tornillo');
  await userEvent.type(screen.getByLabelText('NIT'), '890903938-7');
  await userEvent.type(screen.getByLabelText('Su nombre'), 'Ana Gómez');
  await userEvent.type(screen.getByLabelText('Correo'), 'ana@eltornillo.co');
  await userEvent.type(screen.getByLabelText('Contraseña'), password);
  await userEvent.click(screen.getByRole('button', { name: 'Crear cuenta' }));
}

describe('SignUpPage', () => {
  it('exige todos los campos y una contraseña de al menos 8 caracteres', async () => {
    const api = fakeApi(signedOut());
    renderApp('/registro');

    await userEvent.click(await screen.findByRole('button', { name: 'Crear cuenta' }));

    expect(screen.getByLabelText('Nombre de la empresa')).toHaveAttribute('aria-invalid', 'true');
    expect(screen.getByLabelText('NIT')).toHaveAttribute('aria-invalid', 'true');
    expect(screen.getByLabelText('Su nombre')).toHaveAttribute('aria-invalid', 'true');
    expect(screen.getByLabelText('Correo')).toHaveAttribute('aria-invalid', 'true');

    await userEvent.type(screen.getByLabelText('Contraseña'), 'corta');
    await userEvent.click(screen.getByRole('button', { name: 'Crear cuenta' }));

    expect(screen.getByLabelText('Contraseña')).toHaveAccessibleDescription(
      expect.stringContaining('La contraseña debe tener al menos 8 caracteres'),
    );
    expect(api.calls('POST /api/v1/auth/signup')).toHaveLength(0);
  });

  it('muestra en el campo NIT el error de dígito de verificación que devuelve la API', async () => {
    fakeApi({
      ...signedOut(),
      'POST /api/v1/auth/signup': apiError(
        422,
        'invalid_nit_check_digit',
        'El dígito de verificación del NIT no coincide.',
      ),
    });
    renderApp('/registro');

    await fillSignUp();

    await waitFor(() =>
      expect(screen.getByLabelText('NIT')).toHaveAccessibleDescription(
        expect.stringContaining('El dígito de verificación del NIT no coincide.'),
      ),
    );
    expect(screen.getByLabelText('NIT')).toHaveFocus();
  });

  it('registra la empresa y entra a la aplicación', async () => {
    const api = fakeApi({ ...signedOut(), 'POST /api/v1/auth/signup': json(201, anAuthSession()) });
    const { router } = renderApp('/registro');

    await fillSignUp();

    await waitFor(() => expect(router.state.location.pathname).toBe('/'));
    expect(await screen.findByText('Ferretería El Tornillo')).toBeInTheDocument();
    expect(await api.calls('POST /api/v1/auth/signup')[0]?.json()).toEqual({
      tenant_name: 'Ferretería El Tornillo',
      nit: '890903938-7',
      name: 'Ana Gómez',
      email: 'ana@eltornillo.co',
      password: 'clave-segura',
    });
  });
});
