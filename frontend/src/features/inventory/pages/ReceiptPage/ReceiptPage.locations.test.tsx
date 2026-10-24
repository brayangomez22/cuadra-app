import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser, apiError, json, signedIn } from '@/test/fakeApi';
import { aProduct, ids } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { receiptApi } from '../../testing/receiptApi';
import { pickLocation, pickProduct, register } from '../../testing/receiptSteps';

describe('ReceiptPage', () => {
  it('no ofrece sedes inactivas', async () => {
    receiptApi();
    renderApp('/inventario/entrada');

    await userEvent.click(await screen.findByRole('combobox', { name: 'Sede' }));

    expect(await screen.findByRole('option', { name: 'Sede norte' })).toBeInTheDocument();
    expect(screen.queryByRole('option', { name: 'Bodega cerrada' })).not.toBeInTheDocument();
  });

  it('muestra en el campo el error de sede inactiva', async () => {
    receiptApi({
      'POST /api/v1/inventory/receipts': apiError(422, {
        code: 'location_inactive',
        message: 'La sede Bodega principal está inactiva',
      }),
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
