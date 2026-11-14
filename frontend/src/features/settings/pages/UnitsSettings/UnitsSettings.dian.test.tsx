import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { json } from '@/test/fakeApi';
import { page, unitOf, units } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { unitsApi } from '../../testing/settingsApi';

// The meter of a hardware store has its UN/ECE code; the box has none.
const meters = { ...unitOf('m'), dian_code: 'MTR' };
const withCodes = units.map((u) => (u.id === meters.id ? meters : u));

describe('UnitsSettings: código DIAN', () => {
  it('muestra el código DIAN de cada unidad o el de unidad (94) si no tiene', async () => {
    unitsApi({ 'GET /api/v1/units': json(200, page(withCodes)) });
    renderApp('/configuracion/unidades');

    const meter = within((await screen.findByRole('cell', { name: 'Metro' })).closest('tr')!);
    expect(meter.getByRole('cell', { name: 'MTR' })).toBeInTheDocument();
    const box = within(screen.getByRole('cell', { name: 'Caja' }).closest('tr')!);
    expect(box.getByRole('cell', { name: '94 (unidad)' })).toBeInTheDocument();
  });

  it('agrega una unidad con su código DIAN en mayúsculas (almacén de calzado: el par)', async () => {
    const pair = { id: '01920000-0000-7000-8000-000000000509', code: 'par', name: 'Par', dian_code: 'PR' };
    const api = unitsApi({ 'POST /api/v1/units': json(201, pair) });
    renderApp('/configuracion/unidades');

    await userEvent.click(await screen.findByRole('button', { name: 'Agregar unidad' }));
    const dialog = within(screen.getByRole('dialog', { name: 'Agregar unidad' }));
    await userEvent.type(dialog.getByLabelText('Código'), 'par');
    await userEvent.type(dialog.getByLabelText('Nombre'), 'Par');
    await userEvent.type(dialog.getByLabelText('Código DIAN (opcional)'), 'pr');
    await userEvent.click(dialog.getByRole('button', { name: 'Guardar' }));

    await waitFor(() => expect(api.calls('POST /api/v1/units')).toHaveLength(1));
    expect(await api.calls('POST /api/v1/units')[0]!.json()).toEqual({ code: 'par', name: 'Par', dian_code: 'PR' });
  });

  it('al borrar el código DIAN la unidad queda sin él', async () => {
    const api = unitsApi({
      'GET /api/v1/units': json(200, page(withCodes)),
      [`PUT /api/v1/units/${meters.id}`]: json(200, unitOf('m')),
    });
    renderApp('/configuracion/unidades');

    const row = within((await screen.findByRole('cell', { name: 'Metro' })).closest('tr')!);
    await userEvent.click(row.getByRole('button', { name: 'Editar Metro' }));
    const dialog = within(screen.getByRole('dialog', { name: 'Editar unidad' }));
    expect(dialog.getByLabelText('Código DIAN (opcional)')).toHaveValue('MTR');
    await userEvent.clear(dialog.getByLabelText('Código DIAN (opcional)'));
    await userEvent.click(dialog.getByRole('button', { name: 'Guardar' }));

    await waitFor(() => expect(api.calls(`PUT /api/v1/units/${meters.id}`)).toHaveLength(1));
    expect(await api.calls(`PUT /api/v1/units/${meters.id}`)[0]!.json()).toEqual({ code: 'm', name: 'Metro' });
  });

  it('rechaza un código DIAN con símbolos sin llamar a la API', async () => {
    const api = unitsApi();
    renderApp('/configuracion/unidades');

    await userEvent.click(await screen.findByRole('button', { name: 'Agregar unidad' }));
    const dialog = within(screen.getByRole('dialog', { name: 'Agregar unidad' }));
    await userEvent.type(dialog.getByLabelText('Código'), 'kg');
    await userEvent.type(dialog.getByLabelText('Nombre'), 'Kilogramo');
    await userEvent.type(dialog.getByLabelText('Código DIAN (opcional)'), 'K-G');
    await userEvent.click(dialog.getByRole('button', { name: 'Guardar' }));

    expect(await dialog.findByText(/de 1 a 3 letras o números/)).toBeInTheDocument();
    expect(api.calls('POST /api/v1/units')).toHaveLength(0);
  });
});
