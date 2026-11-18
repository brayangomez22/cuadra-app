import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { apiError, fakeApi, signedOut } from '@/test/fakeApi';
import { renderApp } from '@/test/renderApp';
import { linkToken } from '../../testing/authApi';

const reset = 'POST /api/v1/auth/password-reset';

describe('ResetPasswordPage', () => {
  it('cambia la contraseña y lleva a ingresar con ella', async () => {
    const api = fakeApi({ ...signedOut(), [reset]: new Response(null, { status: 204 }) });
    renderApp(`/restablecer#${linkToken}`);

    await userEvent.type(await screen.findByLabelText('Contraseña nueva'), 'otra-frase-segura');
    await userEvent.click(screen.getByRole('button', { name: 'Guardar contraseña' }));

    await waitFor(() => expect(api.calls(reset)).toHaveLength(1));
    expect(await api.calls(reset)[0]!.json()).toEqual({ token: linkToken, password: 'otra-frase-segura' });
    expect(await screen.findByText(/Ya puede ingresar con su contraseña nueva/)).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Ingresar' })).toHaveAttribute('href', '/ingresar');
  });

  it('un enlace ya usado lo dice', async () => {
    fakeApi({
      ...signedOut(),
      [reset]: apiError(410, { code: 'link_used', message: 'El enlace ya se usó. Pida uno nuevo.' }),
    });
    renderApp(`/restablecer#${linkToken}`);

    await userEvent.type(await screen.findByLabelText('Contraseña nueva'), 'otra-frase-segura');
    await userEvent.click(screen.getByRole('button', { name: 'Guardar contraseña' }));

    expect(await screen.findByText('El enlace ya se usó. Pida uno nuevo.')).toBeInTheDocument();
  });

  it('exige al menos 8 caracteres antes de enviar', async () => {
    const api = fakeApi({ ...signedOut() });
    renderApp(`/restablecer#${linkToken}`);

    await userEvent.type(await screen.findByLabelText('Contraseña nueva'), 'corta');
    await userEvent.click(screen.getByRole('button', { name: 'Guardar contraseña' }));

    expect(screen.getByLabelText('Contraseña nueva')).toHaveAttribute('aria-invalid', 'true');
    expect(api.calls(reset)).toHaveLength(0);
  });
});
