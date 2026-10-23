import { screen, waitFor, within } from '@testing-library/react';
import { ids, plain } from '@/test/fixtures';
import { CEMENT_BARCODE, posApi } from '../../testing/posApi';
import {
  checkout,
  clearPosStorage,
  expectSearchFocused,
  findCartLine,
  posted,
  scan,
  user,
} from '../../testing/posSteps';
import { renderApp } from '@/test/renderApp';

beforeEach(clearPosStorage);

describe('PosPage', () => {
  describe('cobro', () => {
    it('no permite cobrar con pago insuficiente', async () => {
      const api = posApi();
      renderApp('/vender');
      await scan('TUB-PVC-12');
      await findCartLine(/Tubo PVC/);

      await user().keyboard('{F4}');
      const dialog = within(await checkout());
      await user().type(dialog.getByLabelText('Valor del pago 1'), '10000{Enter}');

      expect(plain(dialog.getByRole('alert').textContent)).toContain('Faltan $ 1.900');
      expect(api.calls('POST /api/v1/sales')).toHaveLength(0);
    });

    it('confirma con Enter, envía el carrito y muestra el cambio', async () => {
      const api = posApi();
      renderApp('/vender');
      await scan(CEMENT_BARCODE);
      await findCartLine(/Cemento/);
      await scan(CEMENT_BARCODE);
      await scan('TUB-PVC-12');
      await findCartLine(/Tubo PVC/);

      await user().keyboard('{F4}');
      const dialog = within(await checkout());
      expect(dialog.getByLabelText('Valor del pago 1')).toHaveFocus();
      await user().type(dialog.getByLabelText('Valor del pago 1'), '100000');
      expect(plain(dialog.getByText('Cambio', { selector: 'dt' }).nextElementSibling?.textContent)).toBe('$ 11.940');
      await user().keyboard('{Enter}');

      expect(await screen.findByRole('dialog', { name: 'Venta N.º 15' })).toBeInTheDocument();
      // The change of the API's answer, not the preview's.
      expect(plain(screen.getByRole('dialog').textContent)).toContain('$ 8.100');
      expect(await posted(api)).toEqual({
        location_id: ids.main,
        lines: [
          { variant_id: ids.cement, quantity: '2' },
          { variant_id: ids.pipe, quantity: '1' },
        ],
        payments: [{ method: 'cash', amount: '100000' }],
      });

      expect(screen.getByRole('button', { name: 'Nueva venta' })).toHaveFocus();
      await user().keyboard('{Enter}');
      await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
      expect(screen.queryByRole('listitem', { name: /Cemento/ })).not.toBeInTheDocument();
      await expectSearchFocused();
    });

    it('acepta el valor con puntos de miles, como lo escribe el cajero', async () => {
      const api = posApi();
      renderApp('/vender');
      await scan('TUB-PVC-12');
      await findCartLine(/Tubo PVC/);

      await user().keyboard('{F4}');
      const dialog = within(await checkout());
      await user().type(dialog.getByLabelText('Valor del pago 1'), '20.000');
      expect(plain(dialog.getByText('Cambio', { selector: 'dt' }).nextElementSibling?.textContent)).toBe('$ 8.100');
      await user().keyboard('{Enter}');

      await screen.findByRole('dialog', { name: 'Venta N.º 15' });
      expect((await posted(api)).payments).toEqual([{ method: 'cash', amount: '20000' }]);
    });

    it('muestra cuánto falta mientras se escribe el pago', async () => {
      posApi();
      renderApp('/vender');
      await scan('TUB-PVC-12');
      await findCartLine(/Tubo PVC/);

      await user().keyboard('{F4}');
      const dialog = within(await checkout());
      await user().type(dialog.getByLabelText('Valor del pago 1'), '10000');

      expect(plain(dialog.getByText('Falta', { selector: 'dt' }).nextElementSibling?.textContent)).toBe('$ 1.900');
      expect(dialog.queryByText('Cambio', { selector: 'dt' })).not.toBeInTheDocument();
    });

    it('sin escribir el valor, cobra el total exacto en efectivo', async () => {
      const api = posApi();
      renderApp('/vender');
      await scan('TUB-PVC-12');
      await findCartLine(/Tubo PVC/);

      await user().keyboard('{F4}');
      await checkout();
      await user().keyboard('{Enter}');

      await screen.findByRole('dialog', { name: 'Venta N.º 15' });
      expect((await posted(api)).payments).toEqual([{ method: 'cash', amount: '11900.00' }]);
    });

    it('pago mixto calcula el cambio solo con el efectivo', async () => {
      const api = posApi();
      renderApp('/vender');
      await scan('TUB-PVC-12');
      await findCartLine(/Tubo PVC/);
      await user().keyboard('{F4}');
      const dialog = within(await checkout());

      await user().type(dialog.getByLabelText('Valor del pago 1'), '10000');
      await user().click(dialog.getByRole('button', { name: 'Agregar medio de pago' }));
      // The new payment proposes what is missing.
      expect(dialog.getByLabelText('Valor del pago 2')).toHaveValue('1900');
      expect(dialog.getByRole('combobox', { name: 'Medio de pago 2' })).toHaveTextContent('Tarjeta');

      const cash = dialog.getByLabelText('Valor del pago 1');
      await user().clear(cash);
      await user().type(cash, '20000');
      expect(plain(dialog.getByText('Cambio', { selector: 'dt' }).nextElementSibling?.textContent)).toBe('$ 10.000');
      await user().click(dialog.getByRole('button', { name: 'Confirmar venta' }));

      await screen.findByRole('dialog', { name: 'Venta N.º 15' });
      expect((await posted(api)).payments).toEqual([
        { method: 'cash', amount: '20000' },
        { method: 'card', amount: '1900' },
      ]);
    });

    it('rechaza un sobrepago con medios distintos del efectivo', async () => {
      const api = posApi();
      renderApp('/vender');
      await scan('TUB-PVC-12');
      await findCartLine(/Tubo PVC/);
      await user().keyboard('{F4}');
      const dialog = within(await checkout());

      await user().click(dialog.getByRole('combobox', { name: 'Medio de pago 1' }));
      await user().click(await screen.findByRole('option', { name: 'Nequi' }));
      await user().type(dialog.getByLabelText('Valor del pago 1'), '15000');
      await user().click(dialog.getByRole('button', { name: 'Confirmar venta' }));

      expect(dialog.getByRole('alert')).toHaveTextContent(/Solo el efectivo da cambio/);
      expect(api.calls('POST /api/v1/sales')).toHaveLength(0);
    });
  });
});
