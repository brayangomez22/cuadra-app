import { screen, within } from '@testing-library/react';
import { aUser, json, type Me } from '@/test/fakeApi';
import { aProduct, boxOf12, ids, page, unitOf, withUnits } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { preloadPages } from '@/test/preloadPages';
import { purchasesApi } from '../../testing/purchasesApi';
import {
  choose,
  fillAmounts,
  fillInvoice,
  lineGroup,
  pickProduct,
  registerPurchase,
} from '../../testing/purchaseSteps';

const withUnitConversions = (): Me => ({ ...aUser('admin'), modules: ['unit_conversions'] });
const tubes = { 'GET /api/v1/products': json(200, page([withUnits(aProduct(), [boxOf12])])) };

describe('PurchaseFormPage: unidades de compra', () => {
  beforeAll(preloadPages('/compras/nueva'));

  it('una línea en cajas muestra lo que entra en unidad base y envía unit_id', async () => {
    const api = purchasesApi({ user: withUnitConversions(), routes: tubes });
    renderApp('/compras/nueva');

    await fillInvoice();
    await pickProduct(1, 'tubo', 'Tubo PVC');
    await choose('Unidad', 'caja (12 und)', lineGroup(1));
    await fillAmounts(1, '2', '96000');

    const line = within(lineGroup(1));
    expect(line.getByLabelText('Cantidad')).toHaveAccessibleDescription('Entran 24 und.');
    expect(line.getByLabelText('Costo unitario')).toHaveAccessibleDescription(/por caja/);
    await registerPurchase();

    await screen.findByRole('heading', { name: 'Compra FE 1234' });
    expect((await api.calls('POST /api/v1/purchases')[0]?.json()) as unknown).toMatchObject({
      lines: [{ variant_id: ids.pipe, unit_id: unitOf('caja').id, quantity: '2', unit_cost: '96000' }],
    });
  });

  it('sin el módulo no aparece el selector de unidad', async () => {
    purchasesApi({ routes: tubes });
    renderApp('/compras/nueva');

    await pickProduct(1, 'tubo', 'Tubo PVC');

    expect(within(lineGroup(1)).queryByRole('combobox', { name: 'Unidad' })).not.toBeInTheDocument();
  });
});
