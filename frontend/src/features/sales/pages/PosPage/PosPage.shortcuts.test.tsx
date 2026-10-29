import { screen, waitFor, within } from '@testing-library/react';
import { posApi } from '../../testing/posApi';
import {
  cartLine,
  clearPosStorage,
  expectSearchFocused,
  findCartLine,
  scan,
  searchBox,
  user,
} from '../../testing/posSteps';
import { renderApp } from '@/test/renderApp';

beforeEach(clearPosStorage);

describe('PosPage', () => {
  describe('atajos', () => {
    it('F2 enfoca la búsqueda', async () => {
      posApi();
      renderApp('/vender');
      await scan('TUB-PVC-12');
      await user().click((await findCartLine(/Tubo PVC/)).getByLabelText(/^Cantidad/));

      await user().keyboard('{F2}');

      await expectSearchFocused();
    });

    it('F3 abre la búsqueda de cliente', async () => {
      posApi();
      renderApp('/vender');
      await searchBox();

      await user().keyboard('{F3}');

      expect(await screen.findByRole('dialog', { name: 'Elegir cliente' })).toBeInTheDocument();
      // The dialog hides the rest of the screen from assistive technology.
      expect(screen.getByRole('button', { name: 'Elegir cliente', hidden: true })).toHaveAttribute(
        'aria-keyshortcuts',
        'F3',
      );
    });

    it('F4 no abre el cobro con el carrito vacío', async () => {
      posApi();
      renderApp('/vender');
      await searchBox();

      await user().keyboard('{F4}');

      expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    });

    it('Esc con texto en la búsqueda solo la limpia', async () => {
      posApi();
      renderApp('/vender');
      await scan('TUB-PVC-12');
      await findCartLine(/Tubo PVC/);
      await user().type(await searchBox(), 'cem');

      await user().keyboard('{Escape}');

      expect(await searchBox()).toHaveValue('');
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
      expect(cartLine(/Tubo PVC/)).toBeDefined();
    });

    it('Esc pregunta y solo un clic en «Cancelar venta» la borra', async () => {
      posApi();
      renderApp('/vender');
      await scan('TUB-PVC-12');
      await findCartLine(/Tubo PVC/);

      await user().keyboard('{Escape}');
      const dialog = within(await screen.findByRole('dialog', { name: '¿Cancelar la venta?' }));
      // The safe choice has the focus: a stray Enter keeps the sale.
      expect(dialog.getByRole('button', { name: 'Seguir vendiendo' })).toHaveFocus();
      await user().click(dialog.getByRole('button', { name: 'Cancelar venta' }));

      await waitFor(() => expect(screen.queryByRole('listitem', { name: /Tubo PVC/ })).not.toBeInTheDocument());
      await expectSearchFocused();
    });

    it('un escaneo después de Esc no borra la venta', async () => {
      posApi();
      renderApp('/vender');
      await scan('TUB-PVC-12');
      await findCartLine(/Tubo PVC/);

      await user().keyboard('{Escape}');
      await screen.findByRole('dialog', { name: '¿Cancelar la venta?' });
      // The reader types the code and its Enter while the question is open.
      await user().keyboard('7701234567890{Enter}');

      await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
      expect(cartLine(/Tubo PVC/)).toBeDefined();
      await expectSearchFocused();
    });

    it('Esc en la confirmación conserva la venta', async () => {
      posApi();
      renderApp('/vender');
      await scan('TUB-PVC-12');
      await findCartLine(/Tubo PVC/);

      await user().keyboard('{Escape}');
      await screen.findByRole('dialog', { name: '¿Cancelar la venta?' });
      await user().keyboard('{Escape}');

      await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
      expect(cartLine(/Tubo PVC/)).toBeDefined();
    });
  });
});
