import { cleanup, screen, waitFor } from '@testing-library/react';
import { aUser } from '@/test/fakeApi';
import { ids } from '@/test/fixtures';
import { CEMENT_BARCODE, posApi } from '../../testing/posApi';
import {
  cartLine,
  clearPosStorage,
  expectSearchFocused,
  findCartLine,
  scan,
  summaryValue,
  user,
} from '../../testing/posSteps';
import { renderApp } from '@/test/renderApp';

beforeEach(clearPosStorage);

describe('PosPage', () => {
  it('edita una cantidad decimal', async () => {
    posApi();
    renderApp('/vender');
    await scan('TUB-PVC-12');

    const quantity = (await findCartLine(/Tubo PVC/)).getByLabelText(/^Cantidad/);
    await user().clear(quantity);
    await user().type(quantity, '2,5');

    // 2.5 × 10.000 = 25.000 + IVA 19 %.
    await waitFor(() => expect(summaryValue('Total')).toBe('$ 29.750'));
    expect(summaryValue('IVA')).toBe('$ 4.750');
  });

  it('no deja cobrar con una cantidad inválida', async () => {
    posApi();
    renderApp('/vender');
    await scan('TUB-PVC-12');

    await user().clear((await findCartLine(/Tubo PVC/)).getByLabelText(/^Cantidad/));

    expect(cartLine(/Tubo PVC/).getByLabelText(/^Cantidad/)).toHaveAccessibleDescription(/Escriba la cantidad/);
    expect(screen.getByRole('button', { name: 'Cobrar' })).toBeDisabled();
  });

  it('dice qué corregir cuando una línea no deja cobrar', async () => {
    posApi();
    renderApp('/vender');
    await scan('TUB-PVC-12');
    await scan(CEMENT_BARCODE);

    await user().clear((await findCartLine(/Tubo PVC/)).getByLabelText(/^Cantidad/));

    expect(screen.getByRole('button', { name: 'Cobrar' })).toBeDisabled();
    expect(screen.getByText('Corrija 1 línea para cobrar.')).toBeInTheDocument();
    expect(cartLine(/Tubo PVC/).getByText('Sin total')).toBeInTheDocument();
  });

  it('quita una línea del carrito', async () => {
    posApi();
    renderApp('/vender');
    await scan('TUB-PVC-12');

    await user().click((await findCartLine(/Tubo PVC/)).getByRole('button', { name: /Quitar/ }));

    expect(screen.queryByRole('listitem', { name: /Tubo PVC/ })).not.toBeInTheDocument();
    await expectSearchFocused();
  });

  it('un carrito guardado con la versión anterior se descarta sin error', async () => {
    // The cart before variants, by product and without a version.
    sessionStorage.setItem(
      `cuadra.pos.cart.${aUser('cashier').id}`,
      JSON.stringify({ lines: [{ product: { id: ids.pipe, name: 'Tubo PVC' }, quantity: '1', discount: '' }] }),
    );
    posApi();
    renderApp('/vender');

    await scan('CEM-50');

    expect(await findCartLine(/Cemento/)).toBeDefined();
    expect(screen.getAllByRole('listitem', { name: /./ })).toHaveLength(1);
  });

  it('conserva la venta en curso al salir y volver a la pantalla', async () => {
    posApi();
    renderApp('/vender');
    await scan('TUB-PVC-12');
    await findCartLine(/Tubo PVC/);

    cleanup();
    renderApp('/vender');

    expect(await findCartLine(/Tubo PVC/)).toBeDefined();
  });
});
