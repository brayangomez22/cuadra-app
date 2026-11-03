import { expect, test } from '@playwright/test';
import { newBusiness } from '../support/business';
import { sellInCash } from '../support/sales';
import { signUp } from '../support/signUp';
import { createProduct, expectStock, openCash, receiveStock } from '../support/stock';

// A past sale is found in the day's sales, its receipt is printed again and
// voiding it returns the stock.

const product = { name: 'Camiseta cuello V', sku: 'CAMV', price: '20000' };
const sold = { name: 'Camiseta cuello V (Negro · M)', sku: 'CAMV-NEGRO-M' };

test('reimprime la tirilla de una venta pasada y al anularla devuelve el stock', async ({ page }) => {
  // window.print opens the system dialog: count the calls instead.
  await page.addInitScript(() => {
    window.print = () => {
      (window as unknown as { printed: number }).printed =
        ((window as unknown as { printed?: number }).printed ?? 0) + 1;
    };
  });
  await signUp(page, newBusiness('Ropa y calzado'), 'Ropa y calzado');
  await createProduct(page, product, ['M', 'L', 'Negro']);
  await receiveStock(page, sold, '10');
  await openCash(page);
  // 3 × 20.000 + 19 % VAT.
  const number = await sellInCash(page, sold, '3', '71.400');
  await expectStock(page, 'Principal', sold.name, '7 und');

  await test.step('encontrar la venta y reimprimir la tirilla', async () => {
    await page.goto('/ventas');
    await page
      .getByRole('table', { name: 'Ventas' })
      .getByRole('link', { name: number.replace('N.º ', '') })
      .click();
    const receipt = page.getByRole('article', { name: `Tirilla de la venta ${number}` });
    await expect(receipt).toContainText(sold.name);
    await expect(receipt).toContainText('71.400');
    await page.getByRole('button', { name: 'Imprimir tirilla' }).click();
    expect(await page.evaluate(() => (window as unknown as { printed?: number }).printed)).toBe(1);
  });

  await test.step('anular la venta', async () => {
    await page.getByRole('button', { name: 'Anular venta' }).click();
    const dialog = page.getByRole('dialog', { name: `Anular la venta ${number}` });
    await dialog.getByRole('textbox', { name: 'Motivo' }).fill('El cliente cambió de talla');
    await dialog.getByRole('button', { name: 'Anular venta' }).click();
    await expect(page.getByRole('article', { name: `Tirilla de la venta ${number}` })).toContainText('ANULADA');
  });

  await test.step('el stock vuelve', async () => {
    await expectStock(page, 'Principal', sold.name, '10 und');
  });
});
