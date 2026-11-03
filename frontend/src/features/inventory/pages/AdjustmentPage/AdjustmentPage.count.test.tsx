import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { json } from '@/test/fakeApi';
import { aProduct, ids } from '@/test/fixtures';
import { preloadPages } from '@/test/preloadPages';
import { renderApp } from '@/test/renderApp';
import { aMovement, aStockLevel } from '../../testing/fixtures';
import { receiptApi } from '../../testing/receiptApi';

// 120 und of the pipe in the main location.
const adjustApi = () =>
  receiptApi({
    [`GET /api/v1/locations/${ids.main}/stock`]: json(200, { items: [aStockLevel()], total: 1 }),
    [`GET /api/v1/products/${ids.pipe}`]: json(200, aProduct()),
    'POST /api/v1/inventory/adjustments': json(
      201,
      aMovement({ type: 'adjustment', quantity: '-7', balance_after: '113' }),
    ),
  });

const mode = (name: string) =>
  userEvent.click(within(screen.getByRole('group', { name: 'Movimiento' })).getByRole('radio', { name }));

describe('AdjustmentPage: contar lo que hay', () => {
  beforeAll(preloadPages('/inventario/ajuste'));

  it('dice cuánto hay apenas se eligen el producto y la sede', async () => {
    adjustApi();
    renderApp(`/inventario/ajuste?producto=${ids.pipe}&sede=${ids.main}`);

    expect(await screen.findByText('Bodega principal tiene 120 und.')).toBeInTheDocument();
  });

  it('con lo que se contó calcula la diferencia y la envía', async () => {
    const api = adjustApi();
    renderApp(`/inventario/ajuste?producto=${ids.pipe}&sede=${ids.main}`);
    await screen.findByText('Bodega principal tiene 120 und.');

    await mode('Contar lo que hay');
    await userEvent.type(screen.getByLabelText('Cuántas hay'), '113');
    expect(await screen.findByText('Bodega principal pasa de 120 a 113 und.')).toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: 'Diferencia al contar' }));
    await userEvent.click(screen.getByRole('button', { name: 'Registrar ajuste' }));

    await waitFor(() => expect(api.calls('POST /api/v1/inventory/adjustments')).toHaveLength(1));
    expect(await api.calls('POST /api/v1/inventory/adjustments')[0]?.json()).toMatchObject({
      quantity: '-7',
      reason: 'Diferencia al contar',
    });
  });

  it('si lo contado es lo que ya hay, no hay nada que ajustar', async () => {
    const api = adjustApi();
    renderApp(`/inventario/ajuste?producto=${ids.pipe}&sede=${ids.main}`);
    await screen.findByText('Bodega principal tiene 120 und.');

    await mode('Contar lo que hay');
    await userEvent.type(screen.getByLabelText('Cuántas hay'), '120');
    await userEvent.type(screen.getByLabelText('Motivo'), 'Conteo');
    await userEvent.click(screen.getByRole('button', { name: 'Registrar ajuste' }));

    await waitFor(() =>
      expect(screen.getByLabelText('Cuántas hay')).toHaveAccessibleDescription(
        expect.stringContaining('Ya hay 120 und: no hay nada que ajustar'),
      ),
    );
    expect(api.calls('POST /api/v1/inventory/adjustments')).toHaveLength(0);
  });
});
