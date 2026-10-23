import { screen, waitFor, within } from '@testing-library/react';
import { json } from '@/test/fakeApi';
import { aGarment, page, plain, variantBarcode, variantId } from '@/test/fixtures';
import { CEMENT_BARCODE, posApi } from '../../testing/posApi';
import {
  cartLine,
  checkout,
  clearPosStorage,
  expectSearchFocused,
  findCartLine,
  posted,
  scan,
  searchBox,
  summaryValue,
  user,
} from '../../testing/posSteps';
import { renderApp } from '@/test/renderApp';

beforeEach(clearPosStorage);

describe('PosPage', () => {
  it('escanear el mismo código dos veces suma cantidad', async () => {
    posApi();
    renderApp('/vender');

    await scan(CEMENT_BARCODE);
    await findCartLine(/Cemento gris 50kg/);
    await scan(CEMENT_BARCODE);

    await waitFor(() => expect(cartLine(/Cemento gris 50kg/).getByLabelText(/^Cantidad/)).toHaveValue('2'));
    expect(screen.getAllByRole('listitem', { name: /Cemento/ })).toHaveLength(1);
    expect(summaryValue('Total')).toBe('$ 76.160');
    // The reader can scan the next product at once.
    await expectSearchFocused();
    expect(await searchBox()).toHaveValue('');
  });

  it('agrega con la cantidad antes del código (3*CEM-50)', async () => {
    posApi();
    renderApp('/vender');

    await scan('3*CEM-50');

    expect((await findCartLine(/Cemento/)).getByLabelText(/^Cantidad/)).toHaveValue('3');
  });

  it('busca por nombre y elige entre varios resultados con las flechas', async () => {
    posApi();
    renderApp('/vender');

    await user().type(await searchBox(), 'o');
    expect(await screen.findAllByRole('option')).toHaveLength(2);
    await user().keyboard('{ArrowDown}{ArrowDown}{Enter}');

    await findCartLine(/Cemento gris 50kg/);
    expect(screen.queryByRole('listitem', { name: /Tubo PVC/ })).not.toBeInTheDocument();
  });

  describe('variantes', () => {
    it('escanear el código de una variante agrega esa variante', async () => {
      const api = posApi();
      renderApp('/vender');

      await scan(variantBarcode('M', 'Azul'));

      const line = await findCartLine('Camiseta básica (Azul · M)');
      expect(line.getByText(/CAM-M-AZUL/)).toBeInTheDocument();
      expect(summaryValue('Total')).toBe('$ 29.750');
      await user().keyboard('{F4}');
      await checkout();
      await user().keyboard('{Enter}');
      expect((await posted(api)).lines).toEqual([{ variant_id: variantId('M', 'Azul'), quantity: '1' }]);
    });

    it('escanear el SKU de otra variante del mismo producto la agrega en su propia línea', async () => {
      posApi();
      renderApp('/vender');

      await scan(variantBarcode('M', 'Azul'));
      await findCartLine('Camiseta básica (Azul · M)');
      await scan('cam-l-rojo');
      await scan(variantBarcode('M', 'Azul'));

      expect((await findCartLine('Camiseta básica (Rojo · L)')).getByLabelText(/^Cantidad/)).toHaveValue('1');
      await waitFor(() => expect(cartLine('Camiseta básica (Azul · M)').getByLabelText(/^Cantidad/)).toHaveValue('2'));
      // 2 × 29.750 + 32.130
      expect(summaryValue('Total')).toBe('$ 91.630');
    });

    it('buscar por nombre muestra cada variante con su etiqueta y precio', async () => {
      posApi();
      renderApp('/vender');

      await user().type(await searchBox(), 'camiseta');

      const options = await screen.findAllByRole('option');
      expect(options.map((o) => within(o).getByText(/^Camiseta/).textContent)).toEqual([
        'Camiseta básica (Rojo · S)',
        'Camiseta básica (Azul · S)',
        'Camiseta básica (Rojo · M)',
        'Camiseta básica (Azul · M)',
        'Camiseta básica (Rojo · L)',
        'Camiseta básica (Azul · L)',
      ]);
      expect(plain(options[4]?.textContent)).toContain('$ 32.130');
      await user().keyboard('{ArrowDown}{ArrowDown}{ArrowDown}{ArrowDown}{Enter}');
      await findCartLine('Camiseta básica (Azul · M)');
    });

    it('una variante inactiva no se ofrece', async () => {
      const garment = aGarment();
      garment.variants[3]!.active = false; // M, Azul
      posApi({ extra: { 'GET /api/v1/products': json(200, page([garment])) } });
      renderApp('/vender');

      await scan(variantBarcode('M', 'Azul'));

      expect(
        await screen.findByText(new RegExp(`No encontramos «${variantBarcode('M', 'Azul')}»`)),
      ).toBeInTheDocument();
      expect(screen.queryByRole('listitem', { name: /Camiseta/ })).not.toBeInTheDocument();
    });
  });

  it('avisa cuando no encuentra el código', async () => {
    posApi();
    renderApp('/vender');

    await scan('0000000');

    expect(await screen.findByText(/No encontramos «0000000»/)).toBeInTheDocument();
  });
});
