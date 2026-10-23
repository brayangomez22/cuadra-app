import { screen, waitFor } from '@testing-library/react';
import { onTestFinished } from 'vitest';
import { plain } from '@/test/fixtures';
import { CEMENT_BARCODE, posApi } from '../../testing/posApi';
import { clearPosStorage, expectSearchFocused, findCartLine, scan, user } from '../../testing/posSteps';
import { renderApp } from '@/test/renderApp';

beforeEach(clearPosStorage);

describe('PosPage', () => {
  describe('lo que se ve al agregar y al terminar (critique P2)', () => {
    function announcement() {
      return plain(screen.getByRole('status', { name: 'Último producto agregado' }).textContent);
    }

    it('anuncia el producto agregado con su cantidad y el total', async () => {
      posApi();
      renderApp('/vender');

      await scan(CEMENT_BARCODE);
      await findCartLine(/Cemento/);

      await waitFor(() => expect(announcement()).toBe('Cemento gris 50kg: 1 bulto. Total $ 38.080'));
    });

    it('marca la línea recién agregada y la lleva a la vista', async () => {
      // jsdom has no scrollIntoView: a spy stands in for setup.ts's no-op during
      // this test, which is put back after it (Radix Select needs it).
      const scrolled = vi.spyOn(Element.prototype, 'scrollIntoView').mockImplementation(() => undefined);
      onTestFinished(() => scrolled.mockRestore());
      posApi();
      renderApp('/vender');

      await scan('TUB-PVC-12');
      await findCartLine(/Tubo PVC/);
      await scan(CEMENT_BARCODE);
      await findCartLine(/Cemento/);

      expect(screen.getByRole('listitem', { name: /Cemento/ })).toHaveAttribute('data-state', 'added');
      expect(screen.getByRole('listitem', { name: /Tubo PVC/ })).not.toHaveAttribute('data-state');
      expect(scrolled).toHaveBeenLastCalledWith({ block: 'nearest' });
      await expectSearchFocused();
    });

    it('al escanear otra vez el mismo producto lo vuelve a marcar y anuncia la nueva cantidad', async () => {
      posApi();
      renderApp('/vender');

      await scan(CEMENT_BARCODE);
      await findCartLine(/Cemento/);
      await scan('TUB-PVC-12');
      await findCartLine(/Tubo PVC/);
      await scan(CEMENT_BARCODE);

      await waitFor(() => expect(announcement()).toBe('Cemento gris 50kg: 2 bulto. Total $ 88.060'));
      expect(screen.getByRole('listitem', { name: /Cemento/ })).toHaveAttribute('data-state', 'added');
      expect(screen.getByRole('listitem', { name: /Tubo PVC/ })).not.toHaveAttribute('data-state');
    });

    it('la marca desaparece sola después de un momento', async () => {
      posApi();
      renderApp('/vender');

      await scan(CEMENT_BARCODE);
      const line = await screen.findByRole('listitem', { name: /Cemento/ });
      expect(line).toHaveAttribute('data-state', 'added');

      await waitFor(() => expect(line).not.toHaveAttribute('data-state'), { timeout: 3000 });
    });

    it('editar la cantidad mientras la línea está marcada no pierde el foco cuando la marca se apaga', async () => {
      posApi();
      renderApp('/vender');
      await scan(CEMENT_BARCODE);
      const line = await findCartLine(/Cemento/);
      const quantity = line.getByLabelText(/^Cantidad/);
      await user().click(quantity);

      await waitFor(() => expect(screen.getByRole('listitem', { name: /Cemento/ })).not.toHaveAttribute('data-state'), {
        timeout: 3000,
      });
      expect(quantity).toHaveFocus();
      await user().type(quantity, '{Backspace}3');
      expect(quantity).toHaveValue('3');
    });
  });
});
