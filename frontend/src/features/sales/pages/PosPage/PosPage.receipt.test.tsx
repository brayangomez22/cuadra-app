import { screen, within } from '@testing-library/react';
import { json } from '@/test/fakeApi';
import { aSale, plain } from '@/test/fixtures';
import { posApi } from '../../testing/posApi';
import { checkout, clearPosStorage, findCartLine, scan, searchBox, user } from '../../testing/posSteps';
import { renderApp } from '@/test/renderApp';

beforeEach(clearPosStorage);

describe('PosPage', () => {
  describe('tirilla', () => {
    // sellPipe charges a pipe in cash and waits for the sale's result.
    async function sellPipe() {
      await scan('TUB-PVC-12');
      await findCartLine(/Tubo PVC/);
      await user().keyboard('{F4}');
      await checkout();
      await user().keyboard('{Enter}');
      return within(await screen.findByRole('dialog', { name: 'Venta N.º 15' }));
    }

    // The printable receipt lives outside the app and is hidden on screen.
    const printed = () => screen.getByRole('article', { name: 'Tirilla de la venta N.º 15', hidden: true });

    it('después de confirmar imprime la tirilla con los datos del negocio y de la venta', async () => {
      const print = vi.spyOn(window, 'print').mockImplementation(() => undefined);
      posApi();
      renderApp('/vender');

      const done = await sellPipe();
      await user().click(done.getByRole('button', { name: 'Imprimir tirilla' }));

      expect(print).toHaveBeenCalledOnce();
      const receipt = within(printed());
      expect(receipt.getByText('NIT 900123456-8')).toBeInTheDocument();
      expect(receipt.getByText('Bodega principal')).toBeInTheDocument();
      expect(plain(receipt.getByText('Total', { selector: 'dt' }).nextElementSibling?.textContent)).toBe('$ 11.900');
      // The next sale is still one Enter away.
      expect(done.getByRole('button', { name: 'Nueva venta' })).toBeInTheDocument();
    });

    it('sin venta confirmada no hay tirilla para imprimir', async () => {
      posApi();
      renderApp('/vender');
      await searchBox();

      expect(screen.queryByRole('article', { hidden: true })).not.toBeInTheDocument();
    });

    it('usa papel de 80 mm por defecto y recuerda el ancho elegido en el equipo', async () => {
      posApi();
      renderApp('/vender');

      const done = await sellPipe();
      expect(printed()).toHaveAttribute('data-paper', '80');
      await user().click(done.getByRole('combobox', { name: 'Papel' }));
      await user().click(await screen.findByRole('option', { name: '58 mm' }));

      expect(printed()).toHaveAttribute('data-paper', '58');
      expect(localStorage.getItem('cuadra.pos.paper')).toBe('58');
    });

    it('lee el ancho guardado en el equipo', async () => {
      localStorage.setItem('cuadra.pos.paper', '58');
      posApi();
      renderApp('/vender');

      const done = await sellPipe();

      expect(done.getByRole('combobox', { name: 'Papel' })).toHaveTextContent('58 mm');
      expect(printed()).toHaveAttribute('data-paper', '58');
    });
  });

  describe('lo que se ve al agregar y al terminar (critique P2)', () => {
    it('con cambio, lo muestra en grande con lo que hay que entregar', async () => {
      posApi();
      renderApp('/vender');
      await scan('TUB-PVC-12');
      await findCartLine(/Tubo PVC/);
      await user().keyboard('{F4}');
      await user().keyboard('{Enter}');

      const done = within(await screen.findByRole('dialog', { name: 'Venta N.º 15' }));
      const change = done.getByText('Entregue de cambio', { selector: 'dt' });
      expect(plain(change.nextElementSibling?.textContent)).toBe('$ 8.100');
      expect(done.queryByText('Pago exacto, sin cambio')).not.toBeInTheDocument();
    });

    it('un pago exacto dice que no hay cambio', async () => {
      posApi({
        extra: {
          'POST /api/v1/sales': json(
            201,
            aSale({ paid: '11900.00', change: '0.00', payments: [{ method: 'cash', amount: '11900.00' }] }),
          ),
        },
      });
      renderApp('/vender');
      await scan('TUB-PVC-12');
      await findCartLine(/Tubo PVC/);
      await user().keyboard('{F4}');
      await user().keyboard('{Enter}');

      const done = within(await screen.findByRole('dialog', { name: 'Venta N.º 15' }));
      expect(done.getByText('Pago exacto, sin cambio')).toBeInTheDocument();
      expect(done.queryByText('Entregue de cambio')).not.toBeInTheDocument();
    });

    it('muestra el desglose de una venta con pago mixto', async () => {
      posApi({
        extra: {
          'POST /api/v1/sales': json(
            201,
            aSale({
              paid: '21900.00',
              change: '10000.00',
              payments: [
                { method: 'cash', amount: '20000.00' },
                { method: 'card', amount: '1900.00' },
              ],
            }),
          ),
        },
      });
      renderApp('/vender');
      await scan('TUB-PVC-12');
      await findCartLine(/Tubo PVC/);
      await user().keyboard('{F4}');
      await user().keyboard('{Enter}');

      const done = within(await screen.findByRole('dialog', { name: 'Venta N.º 15' }));
      const payments = within(done.getByRole('list', { name: 'Pagos de la venta' }));
      const items = payments.getAllByRole('listitem').map((item) => plain(item.textContent));
      expect(items).toEqual(['Efectivo$ 20.000', 'Tarjeta$ 1.900']);
    });
  });
});
