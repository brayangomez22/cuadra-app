import { expect, test, type Page } from '@playwright/test';
import { newBusiness } from '../support/business';
import { signUp } from '../support/signUp';
import { createProduct, openCash, receiveStock } from '../support/stock';

// A customer from the POS to their receivables: the cashier creates the
// customer on the fly and sells to them in cash (their name on the receipt),
// the owner gives them credit, they buy on credit, pay part of it and the
// rest shows in the aging. A hammer: 10.000 + 19 % IVA = 11.900.

const product = { name: 'Martillo de uña 16 oz', sku: 'MART-16', price: '10000' };
const customer = { name: 'Lucía Rojas', document: '79888777' };

// sellHammer adds a hammer to the cart and opens the checkout.
async function sellHammer(page: Page) {
  await page.getByRole('combobox', { name: 'Buscar producto' }).fill(product.sku);
  await page.getByRole('option', { name: new RegExp(product.name) }).click();
  await page.getByRole('button', { name: /^Cobrar/ }).click();
  return page.getByRole('dialog', { name: 'Cobrar' });
}

// account opens the customer's account from the customer list.
async function account(page: Page) {
  await page.goto('/clientes');
  await page.getByRole('link', { name: `Cuenta de ${customer.name}` }).click();
  await expect(page.getByRole('heading', { name: customer.name, level: 1 })).toBeVisible();
}

test('crea un cliente en el POS, le vende de contado y a crédito, recibe un abono y lo ve en la cartera', async ({
  page,
}) => {
  await test.step('preparar la empresa', async () => {
    await signUp(page, newBusiness('Ferretería'), 'Ferretería');
    await createProduct(page, product);
    await receiveStock(page, product, '10');
    await openCash(page);
  });

  await test.step('crear el cliente al vuelo y venderle de contado', async () => {
    await page.getByRole('button', { name: 'Elegir cliente' }).click();
    const search = page.getByRole('dialog', { name: 'Elegir cliente' });
    await search.getByRole('combobox', { name: 'Buscar cliente' }).fill(customer.document);
    await search.getByRole('button', { name: 'Nuevo cliente' }).click();
    const form = page.getByRole('dialog', { name: 'Nuevo cliente' });
    await expect(form.getByLabel('Número de documento')).toHaveValue(customer.document);
    await form.getByLabel('Nombre', { exact: true }).fill(customer.name);
    await form.getByRole('button', { name: 'Guardar' }).click();
    await expect(page.getByRole('region', { name: 'Cliente de la venta' })).toContainText(customer.name);

    const checkout = await sellHammer(page);
    await checkout.getByRole('button', { name: 'Confirmar venta' }).click();
    const result = page.getByRole('dialog', { name: /^Venta N\.º/ });
    await expect(result).toContainText('11.900');
    // The receipt is for the printer: hidden on screen, with the customer.
    await expect(page.getByRole('article', { name: /^Tirilla de la venta/, includeHidden: true })).toContainText(
      customer.name,
    );
    await result.getByRole('button', { name: 'Nueva venta' }).click();
  });

  await test.step('asignar cupo y plazo', async () => {
    await account(page);
    await page.getByRole('button', { name: 'Cambiar cupo y plazo' }).click();
    await page.getByLabel('Cupo de crédito').fill('100000');
    await page.getByLabel('Plazo en días').fill('30');
    await page.getByRole('button', { name: 'Guardar cupo' }).click();
    await expect(page.getByText('Cupo guardado').first()).toBeVisible();
  });

  await test.step('vender a crédito eligiendo el cliente con F3', async () => {
    await page.goto('/vender');
    await page.getByRole('combobox', { name: 'Buscar producto' }).waitFor();
    await page.keyboard.press('F3');
    const search = page.getByRole('dialog', { name: 'Elegir cliente' });
    await search.getByRole('combobox', { name: 'Buscar cliente' }).fill('lucía');
    await search.getByRole('option', { name: new RegExp(customer.name) }).click();

    const checkout = await sellHammer(page);
    await expect(checkout).toContainText('Crédito disponible');
    await checkout.getByRole('combobox', { name: 'Medio de pago 1' }).click();
    await page.getByRole('option', { name: 'Crédito' }).click();
    await checkout.getByRole('button', { name: 'Confirmar venta' }).click();
    const result = page.getByRole('dialog', { name: /^Venta N\.º/ });
    await expect(result.getByRole('list', { name: 'Pagos de la venta' })).toContainText('Crédito');
    await result.getByRole('button', { name: 'Nueva venta' }).click();
  });

  await test.step('recibir un abono', async () => {
    await account(page);
    const credit = page.getByRole('region', { name: 'Crédito' });
    await expect(credit).toContainText('11.900');
    await page.getByRole('button', { name: 'Registrar abono' }).click();
    const dialog = page.getByRole('dialog', { name: 'Registrar abono' });
    await dialog.getByRole('combobox', { name: 'Medio de pago' }).click();
    await page.getByRole('option', { name: 'Nequi' }).click();
    await dialog.getByLabel('Valor del abono').fill('5000');
    await dialog.getByRole('button', { name: 'Registrar abono' }).click();
    await expect(page.getByText('Queda debiendo $ 6.900').first()).toBeVisible();
    await expect(page.getByRole('table', { name: 'Estado de cuenta' })).toContainText('Abono · Nequi');
  });

  await test.step('ver la cartera por edades', async () => {
    await page.getByRole('link', { name: 'Cartera' }).click();
    const row = page
      .getByRole('table', { name: 'Cartera por edades' })
      .getByRole('row', { name: new RegExp(customer.name) });
    // Due in 30 days: all of it is still to come.
    await expect(row.getByRole('cell').nth(1)).toHaveText('$ 6.900');
  });
});
