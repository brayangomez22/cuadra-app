import { screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser, json, signedIn } from '@/test/fakeApi';
import { aProduct, ids, page, withoutCost } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { receiptApi } from '../../testing/receiptApi';
import { pickLocation, pickProduct, register } from '../../testing/receiptSteps';

describe('ReceiptPage', () => {
  it('registra una entrada y avisa con la existencia nueva', async () => {
    const api = receiptApi();
    renderApp('/inventario/entrada');

    await pickProduct('tubo', 'Tubo PVC presión 1/2 pulgada');
    await pickLocation('Bodega principal');
    await userEvent.type(screen.getByLabelText('Cantidad'), '10');
    const cost = screen.getByLabelText('Costo unitario');
    expect(cost).toHaveValue('8333,33');
    await userEvent.clear(cost);
    await userEvent.type(cost, '8000');
    await register();

    expect(await screen.findByText('Entrada registrada')).toBeInTheDocument();
    expect(screen.getByText(/Bodega principal queda con 130 und/)).toBeInTheDocument();
    expect(await api.calls('POST /api/v1/inventory/receipts')[0]?.json()).toEqual({
      variant_id: ids.pipe,
      location_id: ids.main,
      quantity: '10',
      unit_cost: '8000',
    });
    // Ready for the next product, in the same location.
    expect(screen.getByRole('searchbox', { name: 'Producto' })).toHaveValue('');
    expect(screen.getByLabelText('Cantidad')).toHaveValue('');
  });

  it('no envía una cantidad vacía o de cero', async () => {
    const api = receiptApi();
    renderApp('/inventario/entrada');
    await pickProduct('tubo', 'Tubo PVC presión 1/2 pulgada');
    await pickLocation('Bodega principal');

    await register();
    expect(screen.getByLabelText('Cantidad')).toHaveAccessibleDescription(
      expect.stringContaining('Escriba la cantidad'),
    );

    await userEvent.type(screen.getByLabelText('Cantidad'), '0');
    await register();
    expect(screen.getByLabelText('Cantidad')).toHaveAccessibleDescription(
      expect.stringContaining('La cantidad debe ser mayor que cero'),
    );
    expect(api.calls('POST /api/v1/inventory/receipts')).toHaveLength(0);
  });

  it('exige el producto, la sede y el costo', async () => {
    const api = receiptApi({ ...signedIn(aUser('warehouse')) });
    renderApp('/inventario/entrada');
    await screen.findByRole('searchbox', { name: 'Producto' });

    await userEvent.type(screen.getByLabelText('Cantidad'), '5');
    await register();

    expect(screen.getByRole('searchbox', { name: 'Producto' })).toHaveAccessibleDescription(
      expect.stringContaining('Elija el producto'),
    );
    expect(screen.getByRole('combobox', { name: 'Sede' })).toHaveAccessibleDescription(
      expect.stringContaining('Elija la sede'),
    );
    expect(screen.getByLabelText('Costo unitario')).toHaveAccessibleDescription(
      expect.stringContaining('Escriba el costo unitario'),
    );
    expect(api.calls('POST /api/v1/inventory/receipts')).toHaveLength(0);
  });

  it('el bodeguero escribe el costo porque no ve el del producto', async () => {
    receiptApi({ ...signedIn(aUser('warehouse')), 'GET /api/v1/products': json(200, page([withoutCost(aProduct())])) });
    renderApp('/inventario/entrada');

    await pickProduct('tubo', 'Tubo PVC presión 1/2 pulgada');

    expect(screen.getByLabelText('Costo unitario')).toHaveValue('');
  });

  it('con el lector, Enter elige el producto cuando hay una sola coincidencia', async () => {
    receiptApi();
    renderApp('/inventario/entrada');

    await userEvent.type(await screen.findByRole('searchbox', { name: 'Producto' }), '7701234567890');
    await screen.findByRole('button', { name: /Cemento gris 50kg/ });
    await userEvent.keyboard('{Enter}');

    const chosen = await screen.findByRole('group', { name: 'Producto elegido' });
    expect(within(chosen).getByText('Cemento gris 50kg')).toBeInTheDocument();
    expect(screen.getByLabelText('Cantidad')).toHaveFocus();
  });
});
