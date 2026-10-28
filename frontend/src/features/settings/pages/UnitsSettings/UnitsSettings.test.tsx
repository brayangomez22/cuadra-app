import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { apiError, json } from '@/test/fakeApi';
import { page, unitOf, units } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { unitsApi } from '../../testing/settingsApi';

async function row(name: string) {
  return within((await screen.findByRole('cell', { name })).closest('tr')!);
}

describe('UnitsSettings', () => {
  it('lista las unidades de la empresa con su código', async () => {
    unitsApi();
    renderApp('/configuracion/unidades');

    expect(await screen.findByRole('heading', { name: 'Unidades de medida' })).toBeInTheDocument();
    expect((await row('Metro')).getByRole('cell', { name: 'm' })).toBeInTheDocument();
    expect(screen.getByRole('cell', { name: 'Bulto' })).toBeInTheDocument();
  });

  it('agrega una unidad y la lista se actualiza', async () => {
    // Almacén de ropa: el par; droguería: la caja de blísteres usaría otro código.
    const pair = { id: '01920000-0000-7000-8000-000000000509', code: 'par', name: 'Par' };
    let created = false;
    const api = unitsApi({
      'GET /api/v1/units': () => json(200, page(created ? [...units, pair] : units)),
      'POST /api/v1/units': () => {
        created = true;
        return json(201, pair);
      },
    });
    renderApp('/configuracion/unidades');

    await userEvent.click(await screen.findByRole('button', { name: 'Agregar unidad' }));
    const dialog = within(screen.getByRole('dialog', { name: 'Agregar unidad' }));
    await userEvent.type(dialog.getByLabelText('Código'), 'Par');
    await userEvent.type(dialog.getByLabelText('Nombre'), 'Par');
    await userEvent.click(dialog.getByRole('button', { name: 'Guardar' }));

    expect(await screen.findByRole('cell', { name: 'Par' })).toBeInTheDocument();
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    expect(await api.calls('POST /api/v1/units')[0]!.json()).toEqual({ code: 'Par', name: 'Par' });
  });

  it('muestra en su campo un código repetido', async () => {
    unitsApi({
      'POST /api/v1/units': apiError(409, { code: 'unit_code_taken', message: 'Ya existe una unidad con ese código.' }),
    });
    renderApp('/configuracion/unidades');

    await userEvent.click(await screen.findByRole('button', { name: 'Agregar unidad' }));
    const dialog = within(screen.getByRole('dialog'));
    await userEvent.type(dialog.getByLabelText('Código'), 'm');
    await userEvent.type(dialog.getByLabelText('Nombre'), 'Metro lineal');
    await userEvent.click(dialog.getByRole('button', { name: 'Guardar' }));

    await waitFor(() =>
      expect(dialog.getByLabelText('Código')).toHaveAccessibleDescription(
        expect.stringContaining('Ya existe una unidad con ese código.'),
      ),
    );
  });

  it('edita una unidad con PUT', async () => {
    const meters = unitOf('m');
    const api = unitsApi({ [`PUT /api/v1/units/${meters.id}`]: json(200, { ...meters, name: 'Metro lineal' }) });
    renderApp('/configuracion/unidades');

    await userEvent.click((await row('Metro')).getByRole('button', { name: 'Editar Metro' }));
    const dialog = within(screen.getByRole('dialog', { name: 'Editar unidad' }));
    expect(dialog.getByLabelText('Código')).toHaveValue('m');
    await userEvent.clear(dialog.getByLabelText('Nombre'));
    await userEvent.type(dialog.getByLabelText('Nombre'), 'Metro lineal');
    await userEvent.click(dialog.getByRole('button', { name: 'Guardar' }));

    await waitFor(() => expect(api.calls(`PUT /api/v1/units/${meters.id}`)).toHaveLength(1));
    expect(await api.calls(`PUT /api/v1/units/${meters.id}`)[0]!.json()).toEqual({ code: 'm', name: 'Metro lineal' });
  });

  it('borra una unidad después de confirmar y la quita de la lista', async () => {
    const bulto = unitOf('bulto');
    let deleted = false;
    const api = unitsApi({
      'GET /api/v1/units': () => json(200, page(deleted ? units.filter((u) => u.id !== bulto.id) : units)),
      [`DELETE /api/v1/units/${bulto.id}`]: () => {
        deleted = true;
        return new Response(null, { status: 204 });
      },
    });
    renderApp('/configuracion/unidades');

    await userEvent.click((await row('Bulto')).getByRole('button', { name: 'Borrar Bulto' }));
    expect(api.calls(`DELETE /api/v1/units/${bulto.id}`)).toHaveLength(0);
    await userEvent.click(
      within(screen.getByRole('dialog', { name: '¿Borrar la unidad Bulto?' })).getByRole('button', { name: 'Borrar' }),
    );

    await waitFor(() => expect(api.calls(`DELETE /api/v1/units/${bulto.id}`)).toHaveLength(1));
    await waitFor(() => expect(screen.queryByRole('cell', { name: 'Bulto' })).not.toBeInTheDocument());
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
  });

  it('una unidad en uso no se borra y explica por qué', async () => {
    const meters = unitOf('m');
    unitsApi({
      [`DELETE /api/v1/units/${meters.id}`]: apiError(409, {
        code: 'unit_in_use',
        message: 'Hay productos que usan esta unidad.',
      }),
    });
    renderApp('/configuracion/unidades');

    await userEvent.click((await row('Metro')).getByRole('button', { name: 'Borrar Metro' }));
    const dialog = within(screen.getByRole('dialog'));
    await userEvent.click(dialog.getByRole('button', { name: 'Borrar' }));

    expect(await dialog.findByText('Hay productos que usan esta unidad.')).toBeInTheDocument();
  });
});
