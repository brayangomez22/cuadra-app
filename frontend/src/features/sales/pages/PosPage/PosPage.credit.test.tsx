import { screen, within } from '@testing-library/react';
import { apiError, json } from '@/test/fakeApi';
import { aBuilder, aCredit, customerIds, plain, searchCustomers } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { preloadPages } from '@/test/preloadPages';
import { aSale } from '../../testing/fixtures';
import { posApi } from '../../testing/posApi';
import { checkout, clearPosStorage, findCartLine, posted, scan, user } from '../../testing/posSteps';

beforeAll(preloadPages('/vender'));
beforeEach(clearPosStorage);

type SaleReply = ReturnType<typeof json> | ReturnType<typeof apiError>;

// The POS with the builder, who has 1.150.000 of credit available; the
// sales answer in turn with replies (the sale by default).
function withCredit(replies: SaleReply[] = []) {
  let call = 0;
  return posApi({
    extra: {
      'GET /api/v1/customers': (request) => json(200, { items: searchCustomers([aBuilder()], request), total: 1 }),
      'GET /api/v1/customers/{customerId}/credit': json(200, aCredit()),
      'POST /api/v1/sales': () => replies[call++] ?? json(201, aSale()),
    },
  });
}

const overLimit = () =>
  apiError(409, {
    code: 'credit_limit_exceeded',
    message: 'La venta pasa el cupo disponible del cliente',
    details: { available: '5000.00' },
  });

// chargeTo picks the builder, scans a pipe and opens the checkout.
async function chargeTo() {
  await user().click(await screen.findByRole('button', { name: 'Elegir cliente' }));
  const search = within(await screen.findByRole('dialog', { name: 'Elegir cliente' }));
  await user().type(search.getByRole('combobox', { name: 'Buscar cliente' }), 'carlos');
  await user().click(await search.findByRole('option', { name: /Carlos Pérez/ }));
  await scan('TUB-PVC-12');
  await findCartLine(/Tubo PVC/);
  await user().keyboard('{F4}');
  return checkout();
}

// payOnCredit chooses credit for the only payment and confirms the exact total.
async function payOnCredit(element: HTMLElement) {
  const dialog = within(element);
  await user().click(dialog.getByRole('combobox', { name: 'Medio de pago 1' }));
  await user().click(await screen.findByRole('option', { name: 'Crédito' }));
  await user().click(dialog.getByRole('button', { name: 'Confirmar venta' }));
}

describe('PosPage: venta a crédito', () => {
  it('a consumidor final no le ofrece crédito', async () => {
    withCredit();
    renderApp('/vender');
    await scan('TUB-PVC-12');
    await findCartLine(/Tubo PVC/);
    await user().keyboard('{F4}');
    const dialog = within(await checkout());

    await user().click(dialog.getByRole('combobox', { name: 'Medio de pago 1' }));

    expect(await screen.findByRole('option', { name: 'Efectivo' })).toBeInTheDocument();
    expect(screen.queryByRole('option', { name: 'Crédito' })).not.toBeInTheDocument();
  });

  it('vende a crédito a un cliente con cupo y muestra lo que tiene disponible', async () => {
    const api = withCredit();
    renderApp('/vender');

    const dialog = await chargeTo();
    expect(plain((await within(dialog).findByText('Crédito disponible')).nextElementSibling?.textContent)).toBe(
      '$ 1.150.000',
    );
    await payOnCredit(dialog);

    expect(await screen.findByRole('dialog', { name: 'Venta N.º 15' })).toBeInTheDocument();
    const body = await posted(api);
    expect(body.customer_id).toBe(customerIds.builder);
    expect(body.payments).toEqual([{ method: 'credit', amount: '11900.00' }]);
    expect(body).not.toHaveProperty('credit_authorization');
  });

  it('si pasa el cupo pide la autorización de un administrador y reenvía la venta', async () => {
    const api = withCredit([overLimit()]);
    renderApp('/vender');

    await payOnCredit(await chargeTo());
    const auth = within(await screen.findByRole('dialog', { name: 'Autorizar venta a crédito' }));
    expect(plain(auth.getByText(/Le quedan/).textContent)).toBe('Le quedan $ 5.000 de cupo.');
    await user().type(auth.getByLabelText('Correo del administrador'), 'admin@ferreteria.co');
    await user().type(auth.getByLabelText('Contraseña'), 'clave-segura');
    await user().click(auth.getByRole('button', { name: 'Autorizar y vender' }));

    expect(await screen.findByRole('dialog', { name: 'Venta N.º 15' })).toBeInTheDocument();
    const [, resent] = api.calls('POST /api/v1/sales');
    const body = (await resent!.json()) as Record<string, unknown>;
    expect(body.payments).toEqual([{ method: 'credit', amount: '11900.00' }]);
    expect(body.credit_authorization).toEqual({ email: 'admin@ferreteria.co', password: 'clave-segura' });
  });

  it('una autorización rechazada lo dice sin cerrar el diálogo', async () => {
    const rejected = apiError(403, {
      code: 'credit_authorization_rejected',
      message: 'El correo o la contraseña no son de un usuario que pueda autorizar',
    });
    withCredit([overLimit(), rejected]);
    renderApp('/vender');

    await payOnCredit(await chargeTo());
    const auth = within(await screen.findByRole('dialog', { name: 'Autorizar venta a crédito' }));
    await user().type(auth.getByLabelText('Correo del administrador'), 'cajero@ferreteria.co');
    await user().type(auth.getByLabelText('Contraseña'), 'otra');
    await user().click(auth.getByRole('button', { name: 'Autorizar y vender' }));

    expect(await auth.findByRole('alert')).toHaveTextContent('no son de un usuario que pueda autorizar');
    expect(auth.getByLabelText('Contraseña')).toHaveValue('');
  });
});
