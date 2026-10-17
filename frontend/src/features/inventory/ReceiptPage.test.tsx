import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser, apiError, fakeApi, json, signedIn } from '@/test/fakeApi';
import {
  aCement,
  aGarment,
  aLocation,
  aMovement,
  aProduct,
  ids,
  page,
  query,
  variantBarcode,
  variantId,
  withoutCost,
} from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';

function receiptApi(extra: Parameters<typeof fakeApi>[0] = {}) {
  return fakeApi({
    ...signedIn(),
    'GET /api/v1/locations': json(200, {
      items: [
        aLocation(),
        aLocation({ id: ids.north, name: 'Sede norte' }),
        aLocation({ id: '01920000-0000-7000-8000-000000000299', name: 'Bodega cerrada', active: false }),
      ],
    }),
    'GET /api/v1/products': (request) => {
      const q = query(request).get('q');
      if (q === '7701234567890') return json(200, page([aCement()]));
      if (q === variantBarcode('L', 'Azul') || q === 'camiseta') return json(200, page([aGarment()]));
      return json(200, page([aProduct(), aCement()]));
    },
    'POST /api/v1/inventory/receipts': json(201, aMovement({ quantity: '10', balance_after: '130' })),
    ...extra,
  });
}

async function pickProduct(text: string, name: string) {
  await userEvent.type(await screen.findByRole('searchbox', { name: 'Producto' }), text);
  await userEvent.click(await screen.findByRole('button', { name: new RegExp(name) }));
}

async function pickLocation(name: string) {
  await userEvent.click(screen.getByRole('combobox', { name: 'Sede' }));
  await userEvent.click(await screen.findByRole('option', { name }));
}

const register = () => userEvent.click(screen.getByRole('button', { name: 'Registrar entrada' }));

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
    expect(screen.getByLabelText('Cantidad')).toHaveAccessibleDescription(expect.stringContaining('Escriba la cantidad'));

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

  describe('variantes', () => {
    it('elige una variante, propone su costo y envía su variant_id', async () => {
      const api = receiptApi();
      renderApp('/inventario/entrada');

      await userEvent.type(await screen.findByRole('searchbox', { name: 'Producto' }), 'camiseta');
      expect(await screen.findAllByRole('button', { name: /Camiseta básica \(/ })).toHaveLength(6);
      await userEvent.click(screen.getByRole('button', { name: /Camiseta básica \(Azul · M\)/ }));
      await pickLocation('Bodega principal');
      await userEvent.type(screen.getByLabelText('Cantidad'), '12');
      expect(screen.getByLabelText('Costo unitario')).toHaveValue('15000,00');
      await register();

      expect(await screen.findByText('Entrada registrada')).toBeInTheDocument();
      expect(screen.getByText(/und de Camiseta básica \(Azul · M\)\. Bodega principal queda con/)).toBeInTheDocument();
      expect(await api.calls('POST /api/v1/inventory/receipts')[0]?.json()).toEqual({
        variant_id: variantId('M', 'Azul'),
        location_id: ids.main,
        quantity: '12',
        unit_cost: '15000.00',
      });
    });

    it('con el lector, Enter elige la variante del código escaneado', async () => {
      receiptApi();
      renderApp('/inventario/entrada');

      await userEvent.type(await screen.findByRole('searchbox', { name: 'Producto' }), `${variantBarcode('L', 'Azul')}{Enter}`);

      const chosen = await screen.findByRole('group', { name: 'Producto elegido' });
      expect(within(chosen).getByText('Camiseta básica (Azul · L)')).toBeInTheDocument();
      expect(within(chosen).getByText(/CAM-L-AZUL/)).toBeInTheDocument();
    });

    it('llega con la variante elegida desde el kardex', async () => {
      receiptApi({ [`GET /api/v1/products/${ids.garment}`]: json(200, aGarment()) });
      renderApp(`/inventario/entrada?producto=${ids.garment}&variante=${variantId('L', 'Rojo')}&sede=${ids.north}`);

      const chosen = await screen.findByRole('group', { name: 'Producto elegido' });

      expect(within(chosen).getByText('Camiseta básica (Rojo · L)')).toBeInTheDocument();
      expect(screen.getByLabelText('Costo unitario')).toHaveValue('15000,00');
    });
  });

  it('no ofrece sedes inactivas', async () => {
    receiptApi();
    renderApp('/inventario/entrada');

    await userEvent.click(await screen.findByRole('combobox', { name: 'Sede' }));

    expect(await screen.findByRole('option', { name: 'Sede norte' })).toBeInTheDocument();
    expect(screen.queryByRole('option', { name: 'Bodega cerrada' })).not.toBeInTheDocument();
  });

  it('muestra en el campo el error de sede inactiva', async () => {
    receiptApi({
      'POST /api/v1/inventory/receipts': apiError(422, 'location_inactive', 'La sede Bodega principal está inactiva'),
    });
    renderApp('/inventario/entrada');
    await pickProduct('tubo', 'Tubo PVC presión 1/2 pulgada');
    await pickLocation('Bodega principal');
    await userEvent.type(screen.getByLabelText('Cantidad'), '10');

    await register();

    await waitFor(() =>
      expect(screen.getByRole('combobox', { name: 'Sede' })).toHaveAccessibleDescription(
        expect.stringContaining('La sede Bodega principal está inactiva'),
      ),
    );
  });

  it('llega con el producto y la sede elegidos desde el kardex', async () => {
    receiptApi({ [`GET /api/v1/products/${ids.pipe}`]: json(200, aProduct()) });
    renderApp(`/inventario/entrada?producto=${ids.pipe}&sede=${ids.north}`);

    const chosen = await screen.findByRole('group', { name: 'Producto elegido' });

    expect(within(chosen).getByText('Tubo PVC presión 1/2 pulgada')).toBeInTheDocument();
    await waitFor(() => expect(screen.getByRole('combobox', { name: 'Sede' })).toHaveTextContent('Sede norte'));
  });

  it('el cajero ve que no tiene permiso', async () => {
    receiptApi({ ...signedIn(aUser('cashier')) });
    renderApp('/inventario/entrada');

    expect(await screen.findByRole('heading', { name: 'No tiene permiso para ver esta sección' })).toBeInTheDocument();
  });
});
