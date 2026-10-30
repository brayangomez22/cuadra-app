import { screen, waitFor, within } from '@testing-library/react';
import { apiError } from '@/test/fakeApi';
import { ids, variantBarcode, variantId } from '@/test/fixtures';
import { CEMENT_BARCODE, posApi } from '../../testing/posApi';
import { cartLine, checkout, clearPosStorage, findCartLine, scan, user } from '../../testing/posSteps';
import { renderApp } from '@/test/renderApp';
import { preloadPages } from '@/test/preloadPages';

beforeAll(preloadPages('/vender'));
beforeEach(clearPosStorage);

describe('PosPage', () => {
  describe('variantes', () => {
    it('insufficient_stock marca la línea de la variante, no las demás del producto', async () => {
      posApi({
        extra: {
          'POST /api/v1/sales': apiError(409, {
            code: 'insufficient_stock',
            message: 'No hay stock suficiente de Camiseta básica (Azul · M).',
            details: {
              variant_id: variantId('M', 'Azul'),
              product_id: ids.garment,
            },
          }),
        },
      });
      renderApp('/vender');
      await scan(variantBarcode('M', 'Azul'));
      await findCartLine('Camiseta básica (Azul · M)');
      await scan('CAM-M-ROJO');
      await findCartLine('Camiseta básica (Rojo · M)');
      await user().keyboard('{F4}');
      await checkout();
      await user().keyboard('{Enter}');
      await within(await checkout()).findByRole('alert');
      await user().keyboard('{Escape}');

      await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
      expect(cartLine('Camiseta básica (Azul · M)').getByText('Sin stock suficiente')).toBeInTheDocument();
      expect(cartLine('Camiseta básica (Rojo · M)').queryByText('Sin stock suficiente')).not.toBeInTheDocument();
    });
  });

  describe('cobro', () => {
    it('marca la línea sin stock cuando la API responde insufficient_stock', async () => {
      posApi({
        extra: {
          'POST /api/v1/sales': apiError(409, {
            code: 'insufficient_stock',
            message: 'No hay stock suficiente de Cemento gris 50kg.',
            details: {
              variant_id: ids.cement,
              product_id: ids.cement,
            },
          }),
        },
      });
      renderApp('/vender');
      await scan(CEMENT_BARCODE);
      await scan('TUB-PVC-12');
      await findCartLine(/Tubo PVC/);
      await user().keyboard('{F4}');
      await checkout();
      await user().keyboard('{Enter}');

      expect(await within(await checkout()).findByRole('alert')).toHaveTextContent(
        'No hay stock suficiente de Cemento gris 50kg.',
      );
      // Back to the cart, the line to fix is marked.
      await user().keyboard('{Escape}');
      await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
      expect(cartLine(/Cemento/).getByText('Sin stock suficiente')).toBeInTheDocument();
      expect(cartLine(/Tubo PVC/).queryByText('Sin stock suficiente')).not.toBeInTheDocument();
    });
  });
});
