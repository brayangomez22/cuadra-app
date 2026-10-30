import { screen, waitFor, within } from '@testing-library/react';
import { fakeCamera } from '@/test/camera';
import { renderApp } from '@/test/renderApp';
import { preloadPages } from '@/test/preloadPages';
import { CEMENT_BARCODE, posApi } from '../../testing/posApi';
import { clearPosStorage, expectSearchFocused, findCartLine, searchBox, user } from '../../testing/posSteps';

beforeAll(preloadPages('/vender'));
beforeEach(clearPosStorage);

async function openCamera() {
  await user().click(await screen.findByRole('button', { name: 'Escanear con la cámara' }));
  return screen.findByRole('dialog', { name: 'Escanear con la cámara' });
}

describe('PosPage con la cámara', () => {
  it('escanear con la cámara agrega la variante a la venta y deja el escáner abierto', async () => {
    const camera = fakeCamera();
    posApi();
    renderApp('/vender');

    const scanner = await openCamera();
    camera.show(CEMENT_BARCODE);

    await findCartLine(/Cemento gris 50kg/);
    expect(scanner).toBeInTheDocument();
    expect(within(scanner).getByRole('status')).toHaveTextContent(`Código leído: ${CEMENT_BARCODE}`);
  });

  it('agrega con la cantidad escrita antes de escanear (3*)', async () => {
    const camera = fakeCamera();
    posApi();
    renderApp('/vender');

    await user().type(await searchBox(), '3*');
    await openCamera();
    camera.show(CEMENT_BARCODE);

    expect((await findCartLine(/Cemento/)).getByLabelText(/^Cantidad/)).toHaveValue('3');
  });

  it('un código que no existe cierra el escáner y muestra que no se encontró', async () => {
    const camera = fakeCamera();
    posApi();
    renderApp('/vender');

    await openCamera();
    camera.show('0000000000000');

    expect(await screen.findByText('No encontramos «0000000000000» entre los productos activos.')).toBeInTheDocument();
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    await expectSearchFocused();
  });

  it('un código con varias coincidencias cierra el escáner y deja elegir', async () => {
    const camera = fakeCamera();
    posApi();
    renderApp('/vender');

    await openCamera();
    camera.show('Camiseta');

    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    expect(await screen.findAllByRole('option')).not.toHaveLength(0);
    await expectSearchFocused();
  });
});
