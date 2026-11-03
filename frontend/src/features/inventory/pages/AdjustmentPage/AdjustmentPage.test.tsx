import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser, apiError, json, signedIn } from '@/test/fakeApi';
import { aProduct, ids, query, variantId } from '@/test/fixtures';
import { preloadPages } from '@/test/preloadPages';
import { renderApp } from '@/test/renderApp';
import { aMovement, aStockLevel } from '../../testing/fixtures';
import { receiptApi } from '../../testing/receiptApi';
import { pickLocation, pickProduct } from '../../testing/receiptSteps';

const stock = { [`GET /api/v1/locations/${ids.main}/stock`]: json(200, { items: [aStockLevel()], total: 1 }) };
const adjustApi = (extra: Parameters<typeof receiptApi>[0] = {}) =>
  receiptApi({
    ...stock,
    [`GET /api/v1/products/${ids.pipe}`]: json(200, aProduct()),
    'POST /api/v1/inventory/adjustments': json(
      201,
      aMovement({ type: 'adjustment', quantity: '-3', balance_after: '117' }),
    ),
    ...extra,
  });

async function pickDirection(name: string) {
  await userEvent.click(within(screen.getByRole('group', { name: 'Movimiento' })).getByRole('radio', { name }));
}

const save = () => userEvent.click(screen.getByRole('button', { name: 'Registrar ajuste' }));

describe('AdjustmentPage', () => {
  beforeAll(preloadPages('/inventario/ajuste'));

  it('saca mercancía rota: envía la cantidad negativa y avisa con la existencia nueva', async () => {
    const api = adjustApi();
    renderApp('/inventario/ajuste');

    await pickProduct('tubo', 'Tubo PVC presión 1/2 pulgada');
    await pickLocation('Bodega principal');
    await pickDirection('Sale');
    await userEvent.type(screen.getByLabelText('Cantidad'), '3');
    expect(await screen.findByText('Bodega principal pasa de 120 a 117 und.')).toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: 'Rotura' }));
    expect(screen.getByLabelText('Motivo')).toHaveValue('Rotura');
    await save();

    expect(await screen.findByText('Ajuste registrado')).toBeInTheDocument();
    expect(screen.getByText(/Salen 3 und de Tubo PVC.*Bodega principal queda con 117 und/)).toBeInTheDocument();
    expect(await api.calls('POST /api/v1/inventory/adjustments')[0]?.json()).toEqual({
      variant_id: ids.pipe,
      location_id: ids.main,
      quantity: '-3',
      reason: 'Rotura',
    });
    expect(query(api.calls(`GET /api/v1/locations/${ids.main}/stock`)[0]!).getAll('variant_ids')).toEqual([ids.pipe]);
    expect(screen.getByRole('searchbox', { name: 'Producto' })).toHaveValue('');
    expect(screen.getByLabelText('Motivo')).toHaveValue('');
    // The next adjustment does not inherit the direction of this one.
    expect(screen.getByRole('radio', { name: 'Sale' })).not.toBeChecked();
  });

  it('agrega una variante de una prenda encontrada, con cantidad positiva', async () => {
    const api = adjustApi({
      'POST /api/v1/inventory/adjustments': json(201, aMovement({ quantity: '2', balance_after: '122' })),
    });
    renderApp('/inventario/ajuste');

    await pickProduct('camiseta', 'Camiseta básica \\(Azul · L\\)');
    await pickLocation('Bodega principal');
    await pickDirection('Entra');
    await userEvent.type(screen.getByLabelText('Cantidad'), '2');
    await userEvent.type(screen.getByLabelText('Motivo'), 'Apareció en la vitrina');
    await save();

    await screen.findByText('Ajuste registrado');
    expect(await api.calls('POST /api/v1/inventory/adjustments')[0]?.json()).toMatchObject({
      variant_id: variantId('L', 'Azul'),
      quantity: '2',
    });
  });

  it('exige producto, sede, movimiento, cantidad y motivo', async () => {
    const api = adjustApi();
    renderApp('/inventario/ajuste');
    await screen.findByRole('searchbox', { name: 'Producto' });

    await save();

    expect(screen.getByRole('searchbox', { name: 'Producto' })).toHaveAccessibleDescription(
      expect.stringContaining('Elija el producto'),
    );
    expect(screen.getByRole('combobox', { name: 'Sede' })).toHaveAccessibleDescription(
      expect.stringContaining('Elija la sede'),
    );
    expect(screen.getByRole('group', { name: 'Movimiento' })).toHaveAccessibleDescription(
      expect.stringContaining('Elija si sale, entra o cuenta lo que hay'),
    );
    expect(screen.getByLabelText('Cantidad')).toHaveAccessibleDescription(
      expect.stringContaining('Escriba la cantidad'),
    );
    expect(screen.getByLabelText('Motivo')).toHaveAccessibleDescription(expect.stringContaining('Escriba el motivo'));
    expect(api.calls('POST /api/v1/inventory/adjustments')).toHaveLength(0);
  });

  it('sin stock suficiente marca la cantidad y nombra el producto y la sede', async () => {
    adjustApi({
      'POST /api/v1/inventory/adjustments': apiError(409, {
        code: 'insufficient_stock',
        message: 'No hay stock suficiente.',
      }),
    });
    renderApp(`/inventario/ajuste?producto=${ids.pipe}&sede=${ids.main}`);
    await screen.findByRole('group', { name: 'Producto elegido' });

    await pickDirection('Sale');
    await userEvent.type(screen.getByLabelText('Cantidad'), '500');
    await userEvent.type(screen.getByLabelText('Motivo'), 'Pérdida');
    await save();

    await waitFor(() =>
      expect(screen.getByLabelText('Cantidad')).toHaveAccessibleDescription(
        expect.stringContaining('No hay stock suficiente de Tubo PVC presión 1/2 pulgada en Bodega principal'),
      ),
    );
  });

  it('avisa si la existencia quedaría negativa y dice la unidad de la cantidad', async () => {
    adjustApi();
    renderApp(`/inventario/ajuste?producto=${ids.pipe}&sede=${ids.main}`);
    const chosen = await screen.findByRole('group', { name: 'Producto elegido' });
    expect(within(chosen).getByText('Tubo PVC presión 1/2 pulgada')).toBeInTheDocument();

    await pickDirection('Sale');
    await userEvent.type(screen.getByLabelText('Cantidad'), '125');

    expect(await screen.findByText(/pasa de 120 a -5 und.*quedaría negativa/i)).toBeInTheDocument();
    expect(screen.getByLabelText('Cantidad')).toHaveAccessibleDescription(expect.stringContaining('und'));
  });

  it('el cajero ve que no tiene permiso', async () => {
    adjustApi({ ...signedIn(aUser('cashier')) });
    renderApp('/inventario/ajuste');

    expect(await screen.findByRole('heading', { name: 'No tiene permiso para ver esta sección' })).toBeInTheDocument();
  });
});
