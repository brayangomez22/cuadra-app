import { expect, test } from '@playwright/test';
import { fakeNit, newBusiness } from '../support/business';
import { chooseOption, signUp } from '../support/signUp';
import { createProduct, expectStock, pickProduct, receiveStock } from '../support/stock';

// A product under its minimum shows up in "Por reponer"; buying what the
// screen suggests raises the stock, shows the purchase in the kardex and takes
// the product out of the list.

const product = { name: 'Tornillo drywall 6 x 1', sku: 'TOR-DRY-61', price: '300' };

test('una compra al proveedor repone lo que estaba bajo el mínimo', async ({ page }) => {
  await signUp(page, newBusiness('Ferretería'), 'Ferretería');
  await createProduct(page, product);
  await receiveStock(page, product, '2');

  await test.step('definir el mínimo y el máximo', async () => {
    await page.goto('/inventario');
    await page.getByRole('button', { name: `Mínimo y máximo de ${product.name}` }).click();
    const dialog = page.getByRole('dialog', { name: 'Mínimo y máximo' });
    await dialog.getByLabel('Stock mínimo').fill('5');
    await dialog.getByLabel('Stock máximo').fill('20');
    await dialog.getByRole('button', { name: 'Guardar' }).click();
    await expect(dialog).toBeHidden();
  });

  await test.step('aparece por reponer con la cantidad sugerida', async () => {
    await page.goto('/inventario/reponer');
    const row = page.getByRole('table', { name: 'Por reponer en Principal' }).getByRole('row', { name: /Tornillo/ });
    // 20 (maximum) − 2 (in stock).
    await expect(row).toContainText('18 und');
  });

  await test.step('crear el proveedor', async () => {
    await page.goto('/compras/proveedores/nuevo');
    await page.getByLabel('Nombre o razón social').fill('Distribuidora Tornillos del Centro');
    await page.getByLabel('NIT').fill(fakeNit());
    await page.getByLabel('Plazo de pago (días)').fill('30');
    await page.getByRole('button', { name: 'Guardar proveedor' }).click();
    await expect(page.getByRole('heading', { name: 'Proveedores' })).toBeVisible();
  });

  await test.step('registrar la compra', async () => {
    await page.goto('/compras/nueva');
    await chooseOption(page, 'Proveedor', /Distribuidora Tornillos del Centro/);
    await chooseOption(page, 'Sede', /^Principal$/);
    await page.getByLabel('Número de factura').fill('FV 2048');
    const line = page.getByRole('group', { name: 'Línea 1' });
    await pickProduct(page, product, line);
    await line.getByLabel('Cantidad').fill('18');
    await line.getByLabel('Costo unitario').fill('150');
    await page.getByRole('button', { name: 'Registrar compra' }).click();
    await page
      .getByRole('dialog', { name: 'Registrar esta compra' })
      .getByRole('button', { name: 'Registrar compra' })
      .click();
    await expect(page.getByRole('heading', { name: 'Compra FV 2048' })).toBeVisible();
  });

  await test.step('el stock sube y la compra queda en el kardex', async () => {
    await expectStock(page, 'Principal', product.name, '20 und');
    await page
      .getByRole('table', { name: 'Existencias en Principal' })
      .getByRole('link', { name: product.name, exact: true })
      .click();
    const movements = page.getByRole('table', { name: `Movimientos de ${product.name}` });
    // The purchase's entry, 18 more for a balance of 20, takes to its invoice.
    await movements
      .getByRole('row', { name: /\+18 20/ })
      .getByRole('link', { name: 'Entrada de mercancía' })
      .click();
    await expect(page.getByRole('heading', { name: 'Compra FV 2048' })).toBeVisible();
  });

  await test.step('ya no está por reponer', async () => {
    await page.goto('/inventario/reponer');
    await expect(page.getByRole('heading', { name: 'Nada por reponer en Principal' })).toBeVisible();
  });
});
