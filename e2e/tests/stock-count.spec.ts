import { expect, test } from '@playwright/test';
import { newBusiness } from '../support/business';
import { signUp } from '../support/signUp';
import { createProduct, expectStock, pickProduct, receiveStock } from '../support/stock';

// A physical count of a business with variants: one size is short, the count
// is approved and the stock becomes what was counted.

const product = { name: 'Pantalón dril', sku: 'PDRIL', price: '60000' };
const counted = { name: 'Pantalón dril (Negro · M)', sku: 'PDRIL-NEGRO-M' };

test('la toma física ajusta el stock a lo contado', async ({ page }) => {
  await signUp(page, newBusiness('Ropa y calzado'), 'Ropa y calzado');
  await createProduct(page, product, ['M', 'L', 'Negro']);
  await receiveStock(page, counted, '10');

  await test.step('iniciar el conteo de la sede', async () => {
    await page.goto('/inventario/conteos');
    await page.getByRole('button', { name: 'Iniciar conteo' }).click();
    await page
      .getByRole('dialog', { name: 'Iniciar conteo en Principal' })
      .getByRole('button', { name: 'Iniciar conteo' })
      .click();
    await expect(page.getByRole('heading', { name: 'Conteo de Principal' })).toBeVisible();
  });

  await test.step('contar dos unidades menos', async () => {
    await pickProduct(page, counted);
    await page.getByRole('textbox', { name: 'Cantidad contada' }).fill('8');
    await page.getByRole('button', { name: 'Registrar' }).click();
    const mine = page.getByRole('list', { name: 'Lo contado en este conteo' });
    // The quantity and its unit are joined by a no-break space.
    await expect(mine.getByRole('listitem').filter({ hasText: counted.name })).toContainText(/8\sund/);
  });

  await test.step('revisar las diferencias y aprobar', async () => {
    await page.goto('/inventario/conteos');
    await page.getByRole('region', { name: 'Conteo abierto' }).getByRole('link', { name: 'Ver diferencias' }).click();
    const row = page.getByRole('table', { name: 'Diferencias del conteo' }).getByRole('row', { name: /Negro · M/ });
    await expect(row).toContainText('10');
    await expect(row).toContainText('8');
    await page.getByRole('button', { name: 'Aprobar conteo' }).click();
    await page
      .getByRole('dialog', { name: 'Aprobar el conteo' })
      .getByRole('button', { name: 'Aprobar y ajustar' })
      .click();
    await expect(page.getByRole('heading', { name: 'Toma física' })).toBeVisible();
  });

  await test.step('el stock queda en lo contado', async () => {
    await expectStock(page, 'Principal', counted.name, '8 und');
  });
});
