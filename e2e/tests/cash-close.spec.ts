import { expect, test } from '@playwright/test';
import { newBusiness } from '../support/business';
import { sellInCash } from '../support/sales';
import { signUp } from '../support/signUp';
import { createProduct, openCash, receiveStock } from '../support/stock';

// A cash session from opening to closing: a cash sale, a count that comes up
// short and the closing report with the expected cash and the difference.

const product = { name: 'Candado de bronce 40 mm', sku: 'CAND-40', price: '10000' };

test('cierra la caja con el arqueo y muestra el faltante', async ({ page }) => {
  await signUp(page, newBusiness('Ferretería'), 'Ferretería');
  await createProduct(page, product);
  await receiveStock(page, product, '10');
  await openCash(page);
  // 2 × 10.000 + 19 % VAT.
  await sellInCash(page, product, '2', '23.800');

  await test.step('contar el efectivo y cerrar', async () => {
    await page.goto('/caja/cerrar');
    // Float 100.000 + sale 23.800 = 123.800 expected; 120.000 counted.
    await page.getByLabel('Billetes de $ 100.000').fill('1');
    await page.getByLabel('Billetes de $ 20.000').fill('1');
    await page.getByLabel('Notas (opcional)').fill('Faltan las monedas');
    await page.getByRole('button', { name: 'Cerrar caja' }).click();
    await page.getByRole('dialog').getByRole('button', { name: 'Cerrar caja' }).click();
  });

  await test.step('el reporte muestra lo esperado, lo contado y la diferencia', async () => {
    await expect(page.getByRole('heading', { name: 'Cierre de caja' })).toBeVisible();
    const main = page.getByRole('main');
    await expect(main).toContainText('123.800');
    await expect(main).toContainText('120.000');
    await expect(main).toContainText('3.800');
    await expect(main).toContainText('Faltan las monedas');
  });
});
