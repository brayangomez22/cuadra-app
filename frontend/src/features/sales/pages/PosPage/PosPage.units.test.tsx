import { screen, waitFor } from '@testing-library/react';
import { aUser, type Me } from '@/test/fakeApi';
import { aGarment, aProduct, boxOf12, packOf3, plain, unitOf, variantId, withUnits } from '@/test/fixtures';
import { posApi } from '../../testing/posApi';
import {
  cartLine,
  checkout,
  clearPosStorage,
  findCartLine,
  posted,
  scan,
  summaryValue,
  user,
} from '../../testing/posSteps';
import { renderApp } from '@/test/renderApp';

beforeEach(clearPosStorage);

describe('PosPage', () => {
  describe('unidades alternativas', () => {
    const withUnitConversions = (): Me => ({ ...aUser('cashier'), modules: ['unit_conversions'] });
    const tubes = () => withUnits(aProduct(), [boxOf12]);
    const red = variantId('M', 'Rojo');
    const garments = () => withUnits(aGarment(), [packOf3], red);

    async function chooseUnit(line: ReturnType<typeof cartLine>, unit: string) {
      await user().click(line.getByRole('combobox', { name: /^Unidad/ }));
      await user().click(await screen.findByRole('option', { name: unit }));
    }

    it('escanear el código de una caja agrega una línea de 1 caja a su precio', async () => {
      posApi({ user: withUnitConversions(), products: [tubes()] });
      renderApp('/vender');

      await scan(boxOf12.barcode!);

      const line = await findCartLine(/Tubo PVC/);
      expect(line.getByLabelText(/^Cantidad/)).toHaveValue('1');
      expect(line.getByRole('combobox', { name: /^Unidad/ })).toHaveTextContent('caja');
      expect(summaryValue('Total')).toBe('$ 130.900');
    });

    it('el selector de unidad cambia el precio y el total de la línea', async () => {
      posApi({ user: withUnitConversions(), products: [tubes()] });
      renderApp('/vender');

      await scan('TUB-PVC-12');
      const line = await findCartLine(/Tubo PVC/);
      expect(summaryValue('Total')).toBe('$ 11.900');

      await chooseUnit(line, 'caja (12 und)');

      await waitFor(() => expect(summaryValue('Total')).toBe('$ 130.900'));
      expect(plain(cartLine(/Tubo PVC/).getByText(/\/ caja/).textContent)).toMatch(/\$ 130\.900 \/ caja/);
    });

    it('una venta en unidad alternativa envía unit_id', async () => {
      const api = posApi({ user: withUnitConversions(), products: [garments()] });
      renderApp('/vender');

      await scan(packOf3.barcode!);
      await findCartLine(/Camiseta básica \(Rojo · M\)/);
      await user().keyboard('{F4}');
      await checkout();
      await user().keyboard('{Enter}');

      await waitFor(() => expect(api.calls('POST /api/v1/sales')).toHaveLength(1));
      expect((await posted(api)).lines).toEqual([{ variant_id: red, quantity: '1', unit_id: unitOf('paq').id }]);
    });

    it('sin el módulo no aparece el selector y la línea se vende en la unidad base', async () => {
      posApi({ products: [tubes()] });
      renderApp('/vender');

      await scan('TUB-PVC-12');

      const line = await findCartLine(/Tubo PVC/);
      expect(line.queryByRole('combobox', { name: /^Unidad/ })).not.toBeInTheDocument();
      expect(summaryValue('Total')).toBe('$ 11.900');
    });
  });
});
