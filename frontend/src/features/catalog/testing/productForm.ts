import { screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser, fakeApi, json, signedIn, type Me } from '@/test/fakeApi';
import { axes, ids, page, units } from '@/test/fixtures';

// The categories GET /categories answers.
export const categories = { items: [{ id: ids.plumbing, name: 'Plomería', created_at: '2026-10-01T15:00:00Z' }] };

/** The API of the product form: the tenant's lists, no attributes, and the routes a test adds. */
export function productFormApi(extra: Parameters<typeof fakeApi>[0] = {}, user: Me = aUser()) {
  return fakeApi({
    ...signedIn(user),
    'GET /api/v1/categories': json(200, categories),
    'GET /api/v1/products': json(200, page([])),
    'GET /api/v1/units': json(200, { items: units }),
    'GET /api/v1/product-attributes': json(200, { items: [] }),
    'GET /api/v1/variant-axes': json(200, { items: axes }),
    ...extra,
  });
}

/** The user with the module variants on. */
export const withVariants = (user = aUser()): Me => ({ ...user, modules: ['variants'] });

/** Fills a simple product (a hardware store's pipe); an empty amount is left empty. */
export async function fillProduct({ cost = '8333,33', price = '10000' }: { cost?: string; price?: string } = {}) {
  await userEvent.type(await screen.findByLabelText('SKU'), 'tub-pvc-12');
  await userEvent.type(screen.getByLabelText('Nombre'), 'Tubo PVC presión 1/2 pulgada');
  if (cost) await userEvent.type(screen.getByLabelText('Costo'), cost);
  if (price) await userEvent.type(screen.getByLabelText('Precio de venta (sin IVA)'), price);
}

export const saveProduct = () => userEvent.click(screen.getByRole('button', { name: 'Guardar producto' }));

export async function chooseValues(axis: string, values: string[]) {
  const group = within(await screen.findByRole('group', { name: axis }));
  for (const value of values) await userEvent.click(group.getByRole('checkbox', { name: value }));
}

export async function fillGarment() {
  await userEvent.type(await screen.findByLabelText('SKU'), 'cam');
  await userEvent.type(screen.getByLabelText('Nombre'), 'Camiseta básica');
  await userEvent.type(screen.getByLabelText('Costo'), '15000');
  await userEvent.type(screen.getByLabelText('Precio de venta (sin IVA)'), '25000');
  await chooseValues('Talla', ['S', 'M', 'L']);
  await chooseValues('Color', ['Rojo', 'Azul']);
}
