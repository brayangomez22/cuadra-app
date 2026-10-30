import { screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { plain } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { purchasesApi } from '../../testing/purchasesApi';
import {
  confirmDialog,
  fillAmounts,
  fillInvoice,
  pickProduct,
  submitPurchase,
  preloadPurchasePages,
} from '../../testing/purchaseSteps';

async function fillPurchase() {
  await fillInvoice();
  await pickProduct(1, 'cemento', 'Cemento gris 50kg');
  await fillAmounts(1, '10', '28000');
}

describe('PurchaseFormPage: confirmar y descartar', () => {
  beforeAll(preloadPurchasePages);

  it('antes de registrar muestra el resumen de la compra y «Revisar» no la envía', async () => {
    const api = purchasesApi();
    renderApp('/compras/nueva');

    await fillPurchase();
    await submitPurchase();

    const dialog = within(await confirmDialog());
    expect(dialog.getByText('Distribuidora Andina SAS')).toBeInTheDocument();
    expect(dialog.getByText('FE 1234')).toBeInTheDocument();
    expect(dialog.getByText('Bodega principal')).toBeInTheDocument();
    expect(plain(dialog.getByText('Total').nextElementSibling?.textContent)).toBe('$ 333.200');
    await userEvent.click(dialog.getByRole('button', { name: 'Revisar' }));

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    expect(api.calls('POST /api/v1/purchases')).toHaveLength(0);
    expect(screen.getByLabelText('Número de factura')).toHaveValue('FE 1234');
  });

  it('al salir con una compra a medio escribir pregunta antes de descartarla', async () => {
    purchasesApi();
    const { router } = renderApp('/compras/nueva');

    await fillPurchase();
    await userEvent.click(screen.getByRole('link', { name: 'Cancelar' }));

    const dialog = within(await screen.findByRole('dialog', { name: '¿Descartar la compra?' }));
    await userEvent.click(dialog.getByRole('button', { name: 'Seguir editando' }));
    expect(router.state.location.pathname).toBe('/compras/nueva');
    expect(screen.getByLabelText('Número de factura')).toHaveValue('FE 1234');

    await userEvent.click(screen.getByRole('link', { name: 'Cancelar' }));
    await userEvent.click(
      within(await screen.findByRole('dialog', { name: '¿Descartar la compra?' })).getByRole('button', {
        name: 'Descartar',
      }),
    );
    expect(router.state.location.pathname).toBe('/compras');
  });

  it('sin nada escrito sale sin preguntar', async () => {
    purchasesApi();
    const { router } = renderApp('/compras/nueva');

    await screen.findByRole('combobox', { name: 'Proveedor' });
    await userEvent.click(screen.getByRole('link', { name: 'Cancelar' }));

    expect(router.state.location.pathname).toBe('/compras');
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });
});
