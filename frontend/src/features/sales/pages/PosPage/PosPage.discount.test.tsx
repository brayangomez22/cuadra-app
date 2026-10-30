import { screen } from '@testing-library/react';
import { aUser } from '@/test/fakeApi';
import { ids } from '@/test/fixtures';
import { posApi } from '../../testing/posApi';
import { checkout, clearPosStorage, findCartLine, posted, scan, summaryValue, user } from '../../testing/posSteps';
import { renderApp } from '@/test/renderApp';
import { preloadPages } from '@/test/preloadPages';

beforeAll(preloadPages('/vender'));
beforeEach(clearPosStorage);

describe('PosPage', () => {
  describe('descuento', () => {
    it('un cajero con tope 0 no ve el campo de descuento', async () => {
      posApi({ limit: { unlimited: false, max: '0' } });
      renderApp('/vender');
      await scan('TUB-PVC-12');

      await findCartLine(/Tubo PVC/);

      expect(screen.queryByLabelText(/^Descuento/)).not.toBeInTheDocument();
    });

    it('un cajero con tope 10 % no puede dar 15 %', async () => {
      posApi({ limit: { unlimited: false, max: '0.1' } });
      renderApp('/vender');
      await scan('TUB-PVC-12');
      const discount = (await findCartLine(/Tubo PVC/)).getByLabelText(/^Descuento/);

      await user().type(discount, '15');
      expect(discount).toHaveAccessibleDescription(/Máximo 10 %/);
      expect(screen.getByRole('button', { name: 'Cobrar' })).toBeDisabled();

      await user().clear(discount);
      await user().type(discount, '10');
      // 10.000 − 1.000 = 9.000 + IVA 1.710.
      expect(summaryValue('Total')).toBe('$ 10.710');
      expect(screen.getByRole('button', { name: 'Cobrar' })).toBeEnabled();
    });

    it('un administrador descuenta sin tope y el monto viaja antes de IVA', async () => {
      const api = posApi({ user: aUser('admin'), limit: { unlimited: true, max: '1' } });
      renderApp('/vender');
      await scan('TUB-PVC-12');
      const discount = (await findCartLine(/Tubo PVC/)).getByLabelText(/^Descuento/);

      await user().type(discount, '50');

      expect(summaryValue('Total')).toBe('$ 5.950');
      expect(summaryValue('Descuentos')).toBe('$ 5.000');
      await user().keyboard('{F4}');
      await checkout();
      await user().keyboard('{Enter}');
      await screen.findByRole('dialog', { name: 'Venta N.º 15' });
      expect((await posted(api)).lines).toEqual([{ variant_id: ids.pipe, quantity: '1', discount: '5000.00' }]);
    });
  });
});
