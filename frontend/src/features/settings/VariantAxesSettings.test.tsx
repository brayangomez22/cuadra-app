import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser, apiError, fakeApi, json, signedIn } from '@/test/fakeApi';
import { axes, page } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';

const withVariants = { ...aUser('owner'), business_template: 'ropa_calzado', modules: ['variants'] };

function axesApi(routes: Parameters<typeof fakeApi>[0] = {}) {
  return fakeApi({ ...signedIn(withVariants), 'GET /api/v1/variant-axes': json(200, page(axes)), ...routes });
}

describe('VariantAxesSettings', () => {
  it('lista los ejes con sus valores', async () => {
    axesApi();
    renderApp('/configuracion/ejes');

    const row = within((await screen.findByRole('cell', { name: 'talla' })).closest('tr')!);
    expect(row.getByRole('cell', { name: 'S, M, L' })).toBeInTheDocument();
  });

  it('crea un eje con sus valores, uno por línea', async () => {
    // Droguería: la presentación; papelería usaría el calibre.
    const created = {
      id: '01920000-0000-7000-8000-000000000709',
      name: 'presentación',
      values: ['Caja x 10', 'Caja x 30'],
    };
    const api = axesApi({ 'POST /api/v1/variant-axes': json(201, created) });
    renderApp('/configuracion/ejes');

    await userEvent.click(await screen.findByRole('button', { name: 'Agregar eje' }));
    const dialog = within(screen.getByRole('dialog', { name: 'Agregar eje' }));
    await userEvent.type(dialog.getByLabelText('Nombre'), 'Presentación');
    await userEvent.type(dialog.getByLabelText('Valores'), 'Caja x 10{Enter}Caja x 30');
    await userEvent.click(dialog.getByRole('button', { name: 'Guardar' }));

    await waitFor(() => expect(api.calls('POST /api/v1/variant-axes')).toHaveLength(1));
    expect(await api.calls('POST /api/v1/variant-axes')[0]!.json()).toEqual({
      name: 'Presentación',
      values: ['Caja x 10', 'Caja x 30'],
    });
  });

  it('no envía un eje sin valores', async () => {
    const api = axesApi();
    renderApp('/configuracion/ejes');

    await userEvent.click(await screen.findByRole('button', { name: 'Agregar eje' }));
    const dialog = within(screen.getByRole('dialog'));
    await userEvent.type(dialog.getByLabelText('Nombre'), 'Color');
    await userEvent.click(dialog.getByRole('button', { name: 'Guardar' }));

    expect(dialog.getByLabelText('Valores')).toHaveAttribute('aria-invalid', 'true');
    expect(api.calls('POST /api/v1/variant-axes')).toHaveLength(0);
  });

  it('al editar, el nombre no cambia y quitar un valor en uso muestra el error de la API', async () => {
    const size = axes[0]!;
    const api = axesApi({
      [`PUT /api/v1/variant-axes/${size.id}`]: apiError(409, 'axis_value_in_use', 'Hay variantes con la talla L.'),
    });
    renderApp('/configuracion/ejes');

    const row = within((await screen.findByRole('cell', { name: 'talla' })).closest('tr')!);
    await userEvent.click(row.getByRole('button', { name: 'Editar talla' }));
    const dialog = within(screen.getByRole('dialog', { name: 'Editar eje' }));
    expect(dialog.getByLabelText('Nombre')).toHaveAttribute('readonly');
    await userEvent.clear(dialog.getByLabelText('Valores'));
    await userEvent.type(dialog.getByLabelText('Valores'), 'S{Enter}M');
    await userEvent.click(dialog.getByRole('button', { name: 'Guardar' }));

    expect(await dialog.findByText('Hay variantes con la talla L.')).toBeInTheDocument();
    expect(await api.calls(`PUT /api/v1/variant-axes/${size.id}`)[0]!.json()).toEqual({ values: ['S', 'M'] });
  });

  it('borra un eje después de confirmar', async () => {
    const color = axes[1]!;
    const api = axesApi({ [`DELETE /api/v1/variant-axes/${color.id}`]: new Response(null, { status: 204 }) });
    renderApp('/configuracion/ejes');

    const row = within((await screen.findByRole('cell', { name: 'color' })).closest('tr')!);
    await userEvent.click(row.getByRole('button', { name: 'Borrar color' }));
    await userEvent.click(
      within(screen.getByRole('dialog', { name: '¿Borrar el eje color?' })).getByRole('button', { name: 'Borrar' }),
    );

    await waitFor(() => expect(api.calls(`DELETE /api/v1/variant-axes/${color.id}`)).toHaveLength(1));
  });
});
