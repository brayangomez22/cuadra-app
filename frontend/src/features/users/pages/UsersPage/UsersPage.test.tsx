import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser, apiError, json } from '@/test/fakeApi';
import { renderApp } from '@/test/renderApp';
import { cashier, keeper, linkToken, owner, usersApi } from '../../testing/usersApi';

const rowOf = async (name: string) =>
  within((await screen.findByRole('cell', { name: new RegExp(`^${name}`) })).closest('tr')!);

describe('UsersPage', () => {
  it('lista los usuarios con su rol y estado, y las invitaciones abiertas', async () => {
    usersApi();
    renderApp('/usuarios');

    expect((await rowOf('Luis Pérez')).getByText('Cajero')).toBeInTheDocument();
    const inactive = await rowOf('Marta Ríos');
    expect(inactive.getByText('Bodeguero')).toBeInTheDocument();
    expect(inactive.getByText('Inactivo')).toBeInTheDocument();
    expect(await screen.findByRole('cell', { name: /^Pedro Nuevo/ })).toBeInTheDocument();
  });

  it('desactiva un usuario tras confirmar', async () => {
    const api = usersApi({
      [`POST /api/v1/users/${cashier.id}/deactivate`]: json(200, { ...cashier, active: false }),
    });
    renderApp('/usuarios');

    await userEvent.click((await rowOf('Luis Pérez')).getByRole('button', { name: 'Desactivar' }));
    const dialog = within(screen.getByRole('dialog', { name: '¿Desactivar a Luis Pérez?' }));
    await userEvent.click(dialog.getByRole('button', { name: 'Desactivar' }));

    await waitFor(() => expect(api.calls(`POST /api/v1/users/${cashier.id}/deactivate`)).toHaveLength(1));
    expect(await screen.findByText('Usuario desactivado')).toBeInTheDocument();
  });

  it('reactiva un usuario inactivo', async () => {
    const api = usersApi({ [`POST /api/v1/users/${keeper.id}/activate`]: json(200, { ...keeper, active: true }) });
    renderApp('/usuarios');

    await userEvent.click((await rowOf('Marta Ríos')).getByRole('button', { name: 'Reactivar' }));

    await waitFor(() => expect(api.calls(`POST /api/v1/users/${keeper.id}/activate`)).toHaveLength(1));
  });

  it('cambia el rol de un usuario', async () => {
    const api = usersApi({ [`PATCH /api/v1/users/${cashier.id}`]: json(200, { ...cashier, role: 'admin' }) });
    renderApp('/usuarios');

    await userEvent.click((await rowOf('Luis Pérez')).getByRole('button', { name: 'Cambiar rol' }));
    const dialog = within(screen.getByRole('dialog', { name: 'Rol de Luis Pérez' }));
    await userEvent.click(dialog.getByRole('combobox', { name: 'Rol' }));
    await userEvent.click(await screen.findByRole('option', { name: 'Administrador' }));
    await userEvent.click(dialog.getByRole('button', { name: 'Guardar' }));

    await waitFor(() => expect(api.calls(`PATCH /api/v1/users/${cashier.id}`)).toHaveLength(1));
    expect(await api.calls(`PATCH /api/v1/users/${cashier.id}`)[0]!.json()).toEqual({ role: 'admin' });
  });

  it('el dueño no puede quitarse el rol si es el único: lo dice', async () => {
    usersApi({
      [`PATCH /api/v1/users/${owner.id}`]: apiError(409, {
        code: 'last_owner',
        message: 'La empresa necesita al menos un dueño activo.',
      }),
    });
    renderApp('/usuarios');

    const self = await rowOf('Ana Gómez');
    expect(self.queryByRole('button', { name: 'Desactivar' })).not.toBeInTheDocument();
    await userEvent.click(self.getByRole('button', { name: 'Cambiar rol' }));
    const dialog = within(screen.getByRole('dialog', { name: 'Rol de Ana Gómez' }));
    await userEvent.click(dialog.getByRole('combobox', { name: 'Rol' }));
    await userEvent.click(await screen.findByRole('option', { name: 'Administrador' }));
    await userEvent.click(dialog.getByRole('button', { name: 'Guardar' }));

    expect(await dialog.findByText('La empresa necesita al menos un dueño activo.')).toBeInTheDocument();
  });

  it('genera el enlace para restablecer la contraseña y lo muestra para compartir', async () => {
    usersApi({
      [`POST /api/v1/users/${cashier.id}/password-reset`]: json(201, {
        token: linkToken,
        expires_at: '2026-11-19T15:00:00Z',
      }),
    });
    renderApp('/usuarios');

    await userEvent.click((await rowOf('Luis Pérez')).getByRole('button', { name: 'Enlace de contraseña' }));

    const dialog = within(await screen.findByRole('dialog', { name: 'Enlace de contraseña de Luis Pérez' }));
    expect(dialog.getByLabelText('Enlace')).toHaveValue(`http://localhost:3000/restablecer#${linkToken}`);
    expect(dialog.getByRole('button', { name: 'Copiar enlace' })).toBeInTheDocument();
  });

  it('un administrador no ve acciones sobre el dueño', async () => {
    usersApi({}, aUser('admin'));
    renderApp('/usuarios');

    const ownerRow = await rowOf('Ana Gómez');
    expect(ownerRow.queryByRole('button', { name: 'Cambiar rol' })).not.toBeInTheDocument();
    expect((await rowOf('Luis Pérez')).getByRole('button', { name: 'Cambiar rol' })).toBeInTheDocument();
  });

  it('un cajero no entra a la pantalla', async () => {
    usersApi({}, aUser('cashier'));
    renderApp('/usuarios');

    expect(await screen.findByText(/No tiene permiso/)).toBeInTheDocument();
  });
});
