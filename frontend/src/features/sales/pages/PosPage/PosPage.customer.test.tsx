import { screen, waitFor, within } from '@testing-library/react';
import { json } from '@/test/fakeApi';
import { aBuilder, customerIds, searchCustomers } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { posApi } from '../../testing/posApi';
import { checkout, clearPosStorage, findCartLine, posted, scan, user } from '../../testing/posSteps';

beforeEach(clearPosStorage);

const created = aBuilder({
  id: customerIds.created,
  document_number: '79888777',
  name: 'Lucía Rojas',
  phone: undefined,
});

// The POS with a customer to find and one to create on the fly.
function withCustomers() {
  return posApi({
    extra: {
      'GET /api/v1/customers': (request) => json(200, { items: searchCustomers([aBuilder()], request), total: 1 }),
      'POST /api/v1/customers': json(201, created),
    },
  });
}

const customerRegion = () => within(screen.getByRole('region', { name: 'Cliente de la venta' }));

// openSearch waits for the POS and opens the customer search.
async function openSearch() {
  await user().click(await screen.findByRole('button', { name: 'Elegir cliente' }));
  return within(await screen.findByRole('dialog', { name: 'Elegir cliente' }));
}

// sell charges a pipe in cash and waits for the sale's result.
async function sell() {
  await scan('TUB-PVC-12');
  await findCartLine(/Tubo PVC/);
  await user().keyboard('{F4}');
  await checkout();
  await user().keyboard('{Enter}');
  return within(await screen.findByRole('dialog', { name: 'Venta N.º 15' }));
}

const printed = () => within(screen.getByRole('article', { name: 'Tirilla de la venta N.º 15', hidden: true }));

describe('PosPage: cliente', () => {
  it('vende a consumidor final sin elegir cliente', async () => {
    const api = withCustomers();
    renderApp('/vender');

    expect(await screen.findByText('Consumidor final')).toBeInTheDocument();
    await sell();

    expect(await posted(api)).not.toHaveProperty('customer_id');
    expect(api.calls('GET /api/v1/customers')).toHaveLength(0);
  });

  it('la venta envía el cliente elegido y la tirilla lo muestra', async () => {
    const api = withCustomers();
    renderApp('/vender');

    const dialog = await openSearch();
    await user().type(dialog.getByRole('combobox', { name: 'Buscar cliente' }), 'carlos');
    await user().click(await dialog.findByRole('option', { name: /Carlos Pérez/ }));
    expect(customerRegion().getByText('Carlos Pérez')).toBeInTheDocument();
    expect(customerRegion().getByText('CC 1020304050')).toBeInTheDocument();
    await sell();

    expect((await posted(api)).customer_id).toBe(customerIds.builder);
    expect(printed().getByText('Carlos Pérez')).toBeInTheDocument();
    expect(printed().getByText('CC 1020304050')).toBeInTheDocument();
  });

  it('crear un cliente desde el POS lo deja elegido en la venta', async () => {
    const api = withCustomers();
    renderApp('/vender');

    const search = await openSearch();
    await user().type(search.getByRole('combobox', { name: 'Buscar cliente' }), '79888777');
    await user().click(await search.findByRole('button', { name: 'Nuevo cliente' }));
    const form = within(await screen.findByRole('dialog', { name: 'Nuevo cliente' }));
    // What the cashier searched is the new customer's document.
    expect(form.getByLabelText('Número de documento')).toHaveValue('79888777');
    await user().type(form.getByLabelText('Nombre'), 'Lucía Rojas');
    await user().click(form.getByRole('button', { name: 'Guardar' }));

    await waitFor(() => expect(customerRegion().getByText('Lucía Rojas')).toBeInTheDocument());
    expect(await api.calls('POST /api/v1/customers')[0]!.json()).toEqual({
      person_type: 'natural',
      document_type: 'cc',
      document_number: '79888777',
      name: 'Lucía Rojas',
      tax_responsibilities: ['R-99-PN'],
      active: true,
    });
    await sell();
    expect((await posted(api)).customer_id).toBe(customerIds.created);
  });

  it('el cliente se quita al empezar la siguiente venta', async () => {
    withCustomers();
    renderApp('/vender');

    const dialog = await openSearch();
    await user().type(dialog.getByRole('combobox', { name: 'Buscar cliente' }), 'carlos');
    await user().click(await dialog.findByRole('option', { name: /Carlos Pérez/ }));
    const done = await sell();
    await user().click(done.getByRole('button', { name: 'Nueva venta' }));

    expect(customerRegion().getByText('Consumidor final')).toBeInTheDocument();
  });
});
