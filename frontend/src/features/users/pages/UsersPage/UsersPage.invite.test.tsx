import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { apiError, json } from '@/test/fakeApi';
import { renderApp } from '@/test/renderApp';
import { invitation, linkToken, usersApi } from '../../testing/usersApi';

async function invite({ email, name, role }: { email: string; name: string; role: string }) {
  await userEvent.click(await screen.findByRole('button', { name: 'Invitar usuario' }));
  const dialog = within(screen.getByRole('dialog', { name: 'Invitar usuario' }));
  await userEvent.type(dialog.getByLabelText('Correo'), email);
  await userEvent.type(dialog.getByLabelText('Nombre'), name);
  await userEvent.click(dialog.getByRole('combobox', { name: 'Rol' }));
  await userEvent.click(await screen.findByRole('option', { name: role }));
  await userEvent.click(dialog.getByRole('button', { name: 'Guardar' }));
  return dialog;
}

describe('UsersPage: invitaciones', () => {
  it('invita a un bodeguero y muestra el enlace para compartir', async () => {
    const created = { ...invitation, email: 'bodega@modaandina.co', name: 'Sara Bodega', role: 'warehouse' as const };
    const api = usersApi({
      'POST /api/v1/invitations': json(201, {
        invitation: created,
        link: { token: linkToken, expires_at: created.expires_at },
      }),
    });
    renderApp('/usuarios');

    await invite({ email: 'bodega@modaandina.co', name: 'Sara Bodega', role: 'Bodeguero' });

    await waitFor(() => expect(api.calls('POST /api/v1/invitations')).toHaveLength(1));
    expect(await api.calls('POST /api/v1/invitations')[0]!.json()).toEqual({
      email: 'bodega@modaandina.co',
      name: 'Sara Bodega',
      role: 'warehouse',
    });
    const link = within(await screen.findByRole('dialog', { name: 'Invitación de Sara Bodega' }));
    expect(link.getByLabelText('Enlace')).toHaveValue(`http://localhost:3000/invitacion#${linkToken}`);
  });

  it('un correo que ya es de un usuario se marca en su campo', async () => {
    usersApi({
      'POST /api/v1/invitations': apiError(409, { code: 'email_taken', message: 'Ese correo ya es de un usuario.' }),
    });
    renderApp('/usuarios');

    const dialog = await invite({ email: 'luis@eltornillo.co', name: 'Luis', role: 'Cajero' });

    expect(await dialog.findByText('Ese correo ya es de un usuario.')).toBeInTheDocument();
    expect(dialog.getByLabelText('Correo')).toHaveAttribute('aria-invalid', 'true');
  });

  it('anula una invitación abierta', async () => {
    const api = usersApi({ [`DELETE /api/v1/invitations/${invitation.id}`]: new Response(null, { status: 204 }) });
    renderApp('/usuarios');

    const row = within((await screen.findByRole('cell', { name: /^Pedro Nuevo/ })).closest('tr')!);
    await userEvent.click(row.getByRole('button', { name: 'Anular' }));

    await waitFor(() => expect(api.calls(`DELETE /api/v1/invitations/${invitation.id}`)).toHaveLength(1));
    expect(await screen.findByText('Invitación anulada')).toBeInTheDocument();
  });
});
