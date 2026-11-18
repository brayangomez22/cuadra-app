import { screen, within } from '@testing-library/react';
import { aUser, fakeApi, json, signedIn } from '@/test/fakeApi';
import { aLocation } from '@/test/fixtures';
import { preloadPages } from '@/test/preloadPages';
import { renderApp } from '@/test/renderApp';
import { aSale } from '../../testing/fixtures';
import { clearPosStorage } from '../../testing/posSteps';
import { aVoidedSale, SALE_KEY, saleId, salesApi } from '../../testing/salesApi';

beforeAll(preloadPages('/ventas/x'));
beforeEach(clearPosStorage);

const otherCashier = '01920000-0000-7000-8000-000000000201';
const cashSessionId = '01920000-0000-7000-8000-000000000401';

describe('SaleDetailPage: quién vendió y quién anuló (P143, P150)', () => {
  it('nombra en la tirilla al cajero que vendió, aunque no sea quien la abre', async () => {
    salesApi({ [SALE_KEY]: json(200, aSale({ user_id: otherCashier, user_name: 'Luis Pérez' })) });
    renderApp(`/ventas/${saleId}`);

    const receipt = within(await screen.findByRole('article', { name: 'Tirilla de la venta N.º 15' }));
    expect(receipt.getByText('Luis Pérez')).toBeInTheDocument();
  });

  it('dice quién anuló la venta', async () => {
    salesApi({
      [SALE_KEY]: json(
        200,
        aVoidedSale({ void: { ...aVoidedSale().void!, user_id: otherCashier, user_name: 'Marta Ríos' } }),
      ),
    });
    renderApp(`/ventas/${saleId}`);

    expect(await screen.findByText(/Anulada .* por Marta Ríos/)).toBeInTheDocument();
  });

  it('muestra al lado de la tirilla los datos de la venta, con el enlace al cierre de su caja', async () => {
    salesApi({
      [SALE_KEY]: json(200, aSale({ user_id: otherCashier, user_name: 'Luis Pérez', cash_session_id: cashSessionId })),
    });
    renderApp(`/ventas/${saleId}`);

    const facts = within(await screen.findByRole('region', { name: 'Datos de la venta' }));
    expect(facts.getByText('Luis Pérez')).toBeInTheDocument();
    expect(facts.getByText(aLocation().name)).toBeInTheDocument();
    expect(facts.getByText('Efectivo')).toBeInTheDocument();
    expect(facts.getByRole('link', { name: 'Ver la caja' })).toHaveAttribute('href', `/caja/${cashSessionId}`);
  });

  it('el cajero no ve el enlace a la caja de otro', async () => {
    fakeApi({
      ...signedIn(aUser('cashier')),
      'GET /api/v1/locations': json(200, { items: [aLocation()] }),
      [SALE_KEY]: json(200, aSale({ cash_session_id: cashSessionId })),
    });
    renderApp(`/ventas/${saleId}`);

    const facts = within(await screen.findByRole('region', { name: 'Datos de la venta' }));
    expect(facts.queryByRole('link', { name: 'Ver la caja' })).not.toBeInTheDocument();
  });
});
