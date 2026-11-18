import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser, apiError, fakeApi, signedIn } from '@/test/fakeApi';
import { renderApp } from '@/test/renderApp';

const change = 'POST /api/v1/me/password';

async function openDialog() {
  await userEvent.click((await screen.findAllByRole('button', { name: 'Cambiar contraseña' }))[0]!);
  return within(screen.getByRole('dialog', { name: 'Cambiar contraseña' }));
}

describe('ChangePasswordDialog', () => {
  it('cambia la propia contraseña con la actual', async () => {
    const api = fakeApi({ ...signedIn(aUser('cashier')), [change]: new Response(null, { status: 204 }) });
    renderApp('/');

    const dialog = await openDialog();
    await userEvent.type(dialog.getByLabelText('Contraseña actual'), 'la-de-antes-1');
    await userEvent.type(dialog.getByLabelText('Contraseña nueva'), 'la-nueva-segura');
    await userEvent.click(dialog.getByRole('button', { name: 'Guardar' }));

    await waitFor(() => expect(api.calls(change)).toHaveLength(1));
    expect(await api.calls(change)[0]!.json()).toEqual({
      current_password: 'la-de-antes-1',
      new_password: 'la-nueva-segura',
    });
    expect(await screen.findByText('Contraseña cambiada')).toBeInTheDocument();
  });

  it('la contraseña actual equivocada se marca en su campo', async () => {
    fakeApi({
      ...signedIn(aUser('warehouse')),
      [change]: apiError(422, { code: 'wrong_password', message: 'La contraseña actual no es correcta.' }),
    });
    renderApp('/');

    const dialog = await openDialog();
    await userEvent.type(dialog.getByLabelText('Contraseña actual'), 'no-es-esta');
    await userEvent.type(dialog.getByLabelText('Contraseña nueva'), 'la-nueva-segura');
    await userEvent.click(dialog.getByRole('button', { name: 'Guardar' }));

    expect(await dialog.findByText('La contraseña actual no es correcta.')).toBeInTheDocument();
    expect(dialog.getByLabelText('Contraseña actual')).toHaveAttribute('aria-invalid', 'true');
  });
});
