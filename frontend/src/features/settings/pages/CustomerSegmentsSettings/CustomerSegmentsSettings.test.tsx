import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { apiError, json } from '@/test/fakeApi';
import { segments } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { segmentsApi } from '../../testing/settingsApi';

const rowOf = async (name: string) => within((await screen.findByRole('cell', { name })).closest('tr')!);

describe('CustomerSegmentsSettings', () => {
  it('crea un segmento', async () => {
    // Droguería: los clientes de un convenio.
    const created = { id: '01920000-0000-7000-8000-000000000609', name: 'Convenio EPS' };
    const api = segmentsApi({ 'POST /api/v1/customer-segments': json(201, created) });
    renderApp('/configuracion/segmentos');

    await rowOf('Maestro de obra');
    await userEvent.click(screen.getByRole('button', { name: 'Agregar segmento' }));
    const dialog = within(screen.getByRole('dialog', { name: 'Agregar segmento' }));
    await userEvent.type(dialog.getByLabelText('Nombre'), 'Convenio EPS');
    await userEvent.click(dialog.getByRole('button', { name: 'Guardar' }));

    await waitFor(() => expect(api.calls('POST /api/v1/customer-segments')).toHaveLength(1));
    expect(await api.calls('POST /api/v1/customer-segments')[0]!.json()).toEqual({ name: 'Convenio EPS' });
  });

  it('renombra un segmento y muestra el nombre repetido en su campo', async () => {
    const wholesale = segments[1]!;
    const api = segmentsApi({
      [`PUT /api/v1/customer-segments/${wholesale.id}`]: apiError(409, {
        code: 'customer_segment_name_taken',
        message: 'Ya existe el segmento Maestro de obra.',
      }),
    });
    renderApp('/configuracion/segmentos');

    await userEvent.click((await rowOf('Mayorista')).getByRole('button', { name: 'Editar Mayorista' }));
    const dialog = within(screen.getByRole('dialog', { name: 'Renombrar segmento' }));
    await userEvent.clear(dialog.getByLabelText('Nombre'));
    await userEvent.type(dialog.getByLabelText('Nombre'), 'maestro de obra');
    await userEvent.click(dialog.getByRole('button', { name: 'Guardar' }));

    expect(await dialog.findByText('Ya existe el segmento Maestro de obra.')).toBeInTheDocument();
    expect(dialog.getByLabelText('Nombre')).toHaveAttribute('aria-invalid', 'true');
    expect(await api.calls(`PUT /api/v1/customer-segments/${wholesale.id}`)[0]!.json()).toEqual({
      name: 'maestro de obra',
    });
  });

  it('no borra un segmento con clientes y lo dice', async () => {
    const builders = segments[0]!;
    segmentsApi({
      [`DELETE /api/v1/customer-segments/${builders.id}`]: apiError(409, {
        code: 'customer_segment_in_use',
        message: 'El segmento tiene clientes.',
      }),
    });
    renderApp('/configuracion/segmentos');

    await userEvent.click((await rowOf('Maestro de obra')).getByRole('button', { name: 'Borrar Maestro de obra' }));
    const dialog = within(screen.getByRole('dialog', { name: '¿Borrar el segmento Maestro de obra?' }));
    await userEvent.click(dialog.getByRole('button', { name: 'Borrar' }));

    expect(await dialog.findByText('El segmento tiene clientes.')).toBeInTheDocument();
  });

  it('aparece en el menú de configuración', async () => {
    segmentsApi();
    renderApp('/configuracion/segmentos');

    expect(await screen.findByRole('link', { name: 'Segmentos de cliente' })).toHaveAttribute(
      'href',
      '/configuracion/segmentos',
    );
  });
});
