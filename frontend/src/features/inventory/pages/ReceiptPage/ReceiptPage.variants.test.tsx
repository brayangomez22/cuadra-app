import { screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { json } from '@/test/fakeApi';
import { aGarment, ids, variantBarcode, variantId } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { receiptApi } from '../../testing/receiptApi';
import { pickLocation, register } from '../../testing/receiptSteps';

describe('ReceiptPage', () => {
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

      await userEvent.type(
        await screen.findByRole('searchbox', { name: 'Producto' }),
        `${variantBarcode('L', 'Azul')}{Enter}`,
      );

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
});
