import { screen, within } from '@testing-library/react';
import { plain } from '@/test/fixtures';
import { posApi } from '../../testing/posApi';
import { checkout, clearPosStorage, findCartLine, scan, user } from '../../testing/posSteps';
import { renderApp } from '@/test/renderApp';
import { preloadPages } from '@/test/preloadPages';

beforeAll(preloadPages('/vender'));
beforeEach(clearPosStorage);

async function openCheckout() {
  posApi();
  renderApp('/vender');
  await scan('TUB-PVC-12');
  await findCartLine(/Tubo PVC/);
  await user().keyboard('{F4}');
  return within(await checkout());
}

type Dialog = Awaited<ReturnType<typeof openCheckout>>;

const addPayment = (dialog: Dialog) => user().click(dialog.getByRole('button', { name: 'Agregar medio de pago' }));

describe('PosPage', () => {
  describe('medios de pago', () => {
    it('no ofrece en un pago el medio que ya usa otro', async () => {
      const dialog = await openCheckout();
      await addPayment(dialog);

      await user().click(dialog.getByRole('combobox', { name: 'Medio de pago 2' }));
      const options = (await screen.findAllByRole('option')).map((o) => o.textContent);
      expect(options).toEqual(['Tarjeta', 'Nequi', 'Daviplata', 'Transferencia']);
    });

    it('deja de ofrecer otro pago cuando ya están todos los medios', async () => {
      const dialog = await openCheckout();
      for (let i = 0; i < 4; i++) await addPayment(dialog);

      expect(dialog.getByRole('combobox', { name: 'Medio de pago 5' })).toHaveTextContent('Transferencia');
      expect(dialog.queryByRole('button', { name: 'Agregar medio de pago' })).not.toBeInTheDocument();
    });

    it('no muestra errores de un pago mientras el cajero lo corrige, solo al confirmar de nuevo', async () => {
      const dialog = await openCheckout();
      await user().type(dialog.getByLabelText('Valor del pago 1'), '10000');
      await addPayment(dialog);
      await user().clear(dialog.getByLabelText('Valor del pago 2'));
      await user().click(dialog.getByRole('button', { name: 'Confirmar venta' }));
      expect(dialog.getByText('Escriba el valor')).toBeInTheDocument();

      // A 0 on its way to 0,50 is not an error yet.
      await user().type(dialog.getByLabelText('Valor del pago 2'), '0');
      expect(dialog.queryByText('Escriba el valor')).not.toBeInTheDocument();
      expect(dialog.queryByText('El valor debe ser mayor que cero')).not.toBeInTheDocument();

      await user().click(dialog.getByRole('button', { name: 'Confirmar venta' }));
      expect(dialog.getByText('El valor debe ser mayor que cero')).toBeInTheDocument();
    });

    it('muestra lo recibido y lo que falta junto a confirmar la venta', async () => {
      const dialog = await openCheckout();
      await user().type(dialog.getByLabelText('Valor del pago 1'), '10000');

      const summary = within(dialog.getByRole('group', { name: 'Cuenta del cobro' }));
      expect(plain(summary.getByText('Recibido', { selector: 'dt' }).nextElementSibling?.textContent)).toBe('$ 10.000');
      expect(plain(summary.getByText('Falta', { selector: 'dt' }).nextElementSibling?.textContent)).toBe('$ 1.900');
      expect(summary.getByRole('button', { name: 'Confirmar venta' })).toBeInTheDocument();
    });
  });
});
