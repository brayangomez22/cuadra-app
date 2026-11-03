import { expect, test } from '@playwright/test';
import { newBusiness } from '../support/business';
import { chooseOption, signUp } from '../support/signUp';
import { addLocation, createProduct, expectStock, pickProduct, receiveStock } from '../support/stock';

// The manual stock moves: taking out what broke and moving stock to another
// location.

const product = { name: 'Bombillo LED 9 W', sku: 'BOMB-LED-9', price: '5000' };

test('saca lo roto con un ajuste y traslada a otra sede', async ({ page }) => {
  await signUp(page, newBusiness('Ferretería'), 'Ferretería');
  await createProduct(page, product);
  await receiveStock(page, product, '10');

  await test.step('ajuste por rotura', async () => {
    await page.goto('/inventario/ajuste');
    await pickProduct(page, product);
    await chooseOption(page, 'Sede', /^Principal$/);
    await page.getByRole('group', { name: 'Movimiento' }).getByText('Sale', { exact: true }).click();
    await page.getByLabel('Cantidad').fill('2');
    await page.getByRole('button', { name: 'Rotura' }).click();
    await page.getByRole('button', { name: 'Registrar ajuste' }).click();
    await expect(page.getByText('Ajuste registrado').first()).toBeVisible();
    await expectStock(page, 'Principal', product.name, '8 und');
  });

  await test.step('traslado a otra sede', async () => {
    await addLocation(page, 'Sede norte');
    await page.goto('/inventario/traslado');
    await pickProduct(page, product);
    await chooseOption(page, 'Sede de origen', /^Principal$/);
    await chooseOption(page, 'Sede de destino', /^Sede norte$/);
    await page.getByLabel('Cantidad').fill('3');
    await page.getByRole('button', { name: 'Registrar traslado' }).click();
    await expect(page.getByText(/Principal queda con 5 und y Sede norte con 3 und/).first()).toBeVisible();
  });
});
