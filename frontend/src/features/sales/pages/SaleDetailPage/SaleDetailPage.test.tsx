import { screen, within } from '@testing-library/react';
import { json, aUser, signedIn } from '@/test/fakeApi';
import { aBoutique, plain } from '@/test/fixtures';
import { preloadPages } from '@/test/preloadPages';
import { renderApp } from '@/test/renderApp';
import { aSale } from '../../testing/fixtures';
import { clearPosStorage, user } from '../../testing/posSteps';
import { aVoidedSale, customerId, SALE_KEY, saleId, salesApi } from '../../testing/salesApi';

beforeAll(preloadPages('/ventas/x'));
beforeEach(clearPosStorage);

const CUSTOMER_KEY = `GET /api/v1/customers/${customerId}` as const;
const shownReceipt = async () => within(await screen.findByRole('article', { name: 'Tirilla de la venta N.º 15' }));
const printRoot = () => document.querySelector('[data-print-root] article');

describe('SaleDetailPage', () => {
  it('reimprime la tirilla en el papel elegido', async () => {
    const print = vi.spyOn(window, 'print').mockImplementation(() => {});
    salesApi();
    renderApp(`/ventas/${saleId}`);

    const receipt = await shownReceipt();
    expect(receipt.getByText('Tubo PVC presión 1/2 pulgada')).toBeInTheDocument();
    expect(printRoot()).toHaveAttribute('data-paper', '80');

    await user().click(screen.getByRole('combobox', { name: 'Papel' }));
    await user().click(await screen.findByRole('option', { name: '58 mm' }));
    expect(printRoot()).toHaveAttribute('data-paper', '58');
    expect(localStorage.getItem('cuadra.pos.paper')).toBe('58');

    await user().click(screen.getByRole('button', { name: 'Imprimir tirilla' }));
    expect(print).toHaveBeenCalledOnce();
    print.mockRestore();
  });

  it('nombra al cliente de la venta con su documento', async () => {
    salesApi({
      [SALE_KEY]: json(200, aSale({ customer_id: customerId })),
      [CUSTOMER_KEY]: json(200, aBoutique({ id: customerId })),
    });
    renderApp(`/ventas/${saleId}`);

    const receipt = await shownReceipt();
    expect(receipt.getByText('Confecciones Sol SAS')).toBeInTheDocument();
    expect(receipt.getByText('NIT 901234567-7')).toBeInTheDocument();
  });

  it('si no carga el cliente no imprime la venta como consumidor final', async () => {
    salesApi({
      [SALE_KEY]: json(200, aSale({ customer_id: customerId })),
      [CUSTOMER_KEY]: json(500, { error: { code: 'internal', message: 'Falló' } } as never),
    });
    renderApp(`/ventas/${saleId}`);

    expect(await screen.findByText('No pudimos abrir la venta')).toBeInTheDocument();
    expect(screen.queryByText('Consumidor final')).not.toBeInTheDocument();
    expect(printRoot()).toBeNull();
  });

  it('muestra el cajero si la venta es del usuario actual', async () => {
    salesApi();
    renderApp(`/ventas/${saleId}`);

    expect((await shownReceipt()).getByText('Ana Gómez')).toBeInTheDocument();
  });

  it('omite el cajero si la venta es de otro usuario', async () => {
    salesApi({ [SALE_KEY]: json(200, aSale({ user_id: '01920000-0000-7000-8000-000000000002' })) });
    renderApp(`/ventas/${saleId}`);

    expect((await shownReceipt()).queryByText('Cajero')).not.toBeInTheDocument();
  });

  it('muestra el motivo de una venta anulada y ya no deja anularla', async () => {
    salesApi({ [SALE_KEY]: json(200, aVoidedSale()) });
    renderApp(`/ventas/${saleId}`);

    const receipt = await shownReceipt();
    expect(receipt.getByText('ANULADA')).toBeInTheDocument();
    expect(plain(screen.getByText(/El cliente desistió/).textContent)).toContain('11/10/2026, 11:00 a. m.');
    expect(screen.queryByRole('button', { name: 'Anular venta' })).not.toBeInTheDocument();
  });

  it('el cajero ve la venta pero no puede anularla', async () => {
    salesApi(signedIn(aUser('cashier')));
    renderApp(`/ventas/${saleId}`);

    await shownReceipt();
    expect(screen.queryByRole('button', { name: 'Anular venta' })).not.toBeInTheDocument();
  });

  it('el bodeguero no entra a las ventas', async () => {
    salesApi(signedIn(aUser('warehouse')));
    renderApp(`/ventas/${saleId}`);

    expect(await screen.findByRole('heading', { name: /permiso/i })).toBeInTheDocument();
  });
});
