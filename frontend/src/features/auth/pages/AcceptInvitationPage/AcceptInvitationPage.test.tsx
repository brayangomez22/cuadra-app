import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser, anAuthSession, apiError, fakeApi, json, signedOut } from '@/test/fakeApi';
import { renderApp } from '@/test/renderApp';
import { invitationPreview, linkToken } from '../../testing/authApi';

const lookup = 'POST /api/v1/auth/invitations/lookup';
const accept = 'POST /api/v1/auth/invitations/accept';

describe('AcceptInvitationPage', () => {
  it('muestra la invitación y entra a la empresa con la contraseña elegida', async () => {
    const api = fakeApi({
      ...signedOut(),
      [lookup]: json(200, invitationPreview),
      [accept]: json(200, anAuthSession(aUser('cashier'))),
    });
    renderApp(`/invitacion#${linkToken}`);

    expect(await screen.findByText(/Ferretería El Tornillo/)).toBeInTheDocument();
    expect(screen.getByText(/Cajero/)).toBeInTheDocument();
    expect(screen.getByLabelText('Su nombre')).toHaveValue('Luis Pérez');
    await userEvent.type(screen.getByLabelText('Contraseña'), 'una-frase-segura');
    await userEvent.click(screen.getByRole('button', { name: 'Entrar a la empresa' }));

    await waitFor(() => expect(api.calls(accept)).toHaveLength(1));
    expect(await api.calls(accept)[0]!.json()).toEqual({
      token: linkToken,
      name: 'Luis Pérez',
      password: 'una-frase-segura',
    });
    expect(await api.calls(lookup)[0]!.json()).toEqual({ token: linkToken });
  });

  it('una contraseña muy común se marca en su campo', async () => {
    fakeApi({
      ...signedOut(),
      [lookup]: json(200, invitationPreview),
      [accept]: apiError(422, { code: 'password_too_common', message: 'Esa contraseña es de las más usadas.' }),
    });
    renderApp(`/invitacion#${linkToken}`);

    await userEvent.type(await screen.findByLabelText('Contraseña'), 'contraseña1');
    await userEvent.click(screen.getByRole('button', { name: 'Entrar a la empresa' }));

    expect(await screen.findByText('Esa contraseña es de las más usadas.')).toBeInTheDocument();
    expect(screen.getByLabelText('Contraseña')).toHaveAttribute('aria-invalid', 'true');
  });

  it('un enlace vencido lo dice y no muestra el formulario', async () => {
    fakeApi({
      ...signedOut(),
      [lookup]: apiError(410, { code: 'link_expired', message: 'El enlace venció. Pida uno nuevo.' }),
    });
    renderApp(`/invitacion#${linkToken}`);

    expect(await screen.findByText('El enlace venció. Pida uno nuevo.')).toBeInTheDocument();
    expect(screen.queryByLabelText('Contraseña')).not.toBeInTheDocument();
  });

  it('un enlace sin token dice que está incompleto sin llamar a la API', async () => {
    const api = fakeApi({ ...signedOut() });
    renderApp('/invitacion');

    expect(await screen.findByText(/El enlace está incompleto/)).toBeInTheDocument();
    expect(api.calls(lookup)).toHaveLength(0);
  });
});
