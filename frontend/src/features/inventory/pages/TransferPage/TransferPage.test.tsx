import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser, apiError, json, signedIn } from '@/test/fakeApi';
import { aLocation, ids } from '@/test/fixtures';
import { preloadPages } from '@/test/preloadPages';
import { renderApp } from '@/test/renderApp';
import { aMovement, aStockLevel } from '../../testing/fixtures';
import { receiptApi } from '../../testing/receiptApi';
import { pickProduct } from '../../testing/receiptSteps';

const transferApi = (extra: Parameters<typeof receiptApi>[0] = {}) =>
  receiptApi({
    [`GET /api/v1/locations/${ids.main}/stock`]: json(200, { items: [aStockLevel()], total: 1 }),
    // The destination has no row for the variant yet: it has zero.
    [`GET /api/v1/locations/${ids.north}/stock`]: json(200, { items: [], total: 0 }),
    'POST /api/v1/inventory/transfers': json(201, {
      out: aMovement({ type: 'transfer_out', quantity: '-20', balance_after: '100' }),
      in: aMovement({ type: 'transfer_in', location_id: ids.north, quantity: '20', balance_after: '20' }),
    }),
    ...extra,
  });

async function pick(label: string, name: string) {
  await userEvent.click(screen.getByRole('combobox', { name: label }));
  await userEvent.click(await screen.findByRole('option', { name }));
}

const save = () => userEvent.click(screen.getByRole('button', { name: 'Registrar traslado' }));

describe('TransferPage', () => {
  beforeAll(preloadPages('/inventario/traslado'));

  it('traslada, muestra lo que queda en cada sede y queda listo para el siguiente', async () => {
    const api = transferApi();
    renderApp('/inventario/traslado');

    await pickProduct('tubo', 'Tubo PVC presión 1/2 pulgada');
    await pick('Sede de origen', 'Bodega principal');
    await pick('Sede de destino', 'Sede norte');
    await userEvent.type(screen.getByLabelText('Cantidad'), '20');
    expect(await screen.findByText('Bodega principal pasa de 120 a 100 und.')).toBeInTheDocument();
    expect(await screen.findByText('Sede norte pasa de 0 a 20 und.')).toBeInTheDocument();
    await save();

    expect(await screen.findByText('Traslado registrado')).toBeInTheDocument();
    expect(screen.getByText(/Bodega principal queda con 100 und y Sede norte con 20 und/)).toBeInTheDocument();
    expect(await api.calls('POST /api/v1/inventory/transfers')[0]?.json()).toEqual({
      variant_id: ids.pipe,
      from_location_id: ids.main,
      to_location_id: ids.north,
      quantity: '20',
    });
    expect(screen.getByRole('searchbox', { name: 'Producto' })).toHaveValue('');
    expect(screen.getByRole('combobox', { name: 'Sede de origen' })).toHaveTextContent('Bodega principal');
  });

  it('no deja trasladar a la misma sede', async () => {
    const api = transferApi();
    renderApp(`/inventario/traslado?sede=${ids.main}`);
    await pickProduct('tubo', 'Tubo PVC presión 1/2 pulgada');
    await pick('Sede de destino', 'Bodega principal');
    await userEvent.type(screen.getByLabelText('Cantidad'), '1');

    await save();

    expect(screen.getByRole('combobox', { name: 'Sede de destino' })).toHaveAccessibleDescription(
      expect.stringContaining('La sede de destino debe ser distinta'),
    );
    expect(api.calls('POST /api/v1/inventory/transfers')).toHaveLength(0);
  });

  it('sin stock en el origen marca la cantidad y nombra el producto y la sede', async () => {
    transferApi({
      'POST /api/v1/inventory/transfers': apiError(409, {
        code: 'insufficient_stock',
        message: 'No hay stock suficiente.',
      }),
    });
    renderApp(`/inventario/traslado?sede=${ids.main}`);
    await pickProduct('tubo', 'Tubo PVC presión 1/2 pulgada');
    await pick('Sede de destino', 'Sede norte');
    await userEvent.type(screen.getByLabelText('Cantidad'), '500');

    await save();

    await waitFor(() =>
      expect(screen.getByLabelText('Cantidad')).toHaveAccessibleDescription(
        expect.stringContaining('No hay stock suficiente de Tubo PVC presión 1/2 pulgada en Bodega principal'),
      ),
    );
  });

  it('con una sola sede activa explica que hacen falta dos', async () => {
    transferApi({ 'GET /api/v1/locations': json(200, { items: [aLocation()] }) });
    renderApp('/inventario/traslado');

    expect(await screen.findByText('Hacen falta dos sedes activas')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Registrar traslado' })).not.toBeInTheDocument();
  });

  it('el cajero ve que no tiene permiso', async () => {
    transferApi({ ...signedIn(aUser('cashier')) });
    renderApp('/inventario/traslado');

    expect(await screen.findByRole('heading', { name: 'No tiene permiso para ver esta sección' })).toBeInTheDocument();
  });
});
