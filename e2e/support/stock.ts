import { expect, type Page } from '@playwright/test';
import { chooseOption, createLocation } from './signUp';

type Product = { name: string; sku: string; price: string };

/** Creates a product in units, with a cost of 6.000; values checks those values of its variant axes. */
export async function createProduct(page: Page, product: Product, values: string[] = []) {
  await page.goto('/productos/nuevo');
  await page.getByLabel('SKU', { exact: true }).fill(product.sku);
  await page.getByLabel('Nombre', { exact: true }).fill(product.name);
  await chooseOption(page, 'Unidad', /^Unidad \(und\)$/);
  await page.getByLabel('Costo', { exact: true }).fill('6000');
  await page.getByLabel('Precio de venta (sin IVA)').fill(product.price);
  for (const value of values) {
    await page.getByRole('checkbox', { name: value, exact: true }).check();
  }
  if (values.length > 0) {
    await expect(page.getByRole('table', { name: 'Variantes del producto' }).getByRole('row')).toHaveCount(3);
  }
  await page.getByRole('button', { name: 'Guardar producto' }).click();
  await expect(page).toHaveURL('/productos');
  await expect(page.getByText(product.name).first()).toBeVisible();
}

/** Creates the location Principal and receives quantity of the variant found by its SKU. */
export async function receiveStock(page: Page, variant: { name: string; sku: string }, quantity: string) {
  await createLocation(page, 'Principal');
  await page.getByRole('link', { name: 'Registrar entrada' }).first().click();
  await page.getByLabel('Producto', { exact: true }).fill(variant.sku);
  await page.getByRole('list', { name: 'Resultados' }).getByRole('button', { name: variant.name }).click();
  await page.getByLabel('Cantidad', { exact: true }).fill(quantity);
  await page.getByRole('button', { name: 'Registrar entrada' }).click();
  // The toast is announced too: the same text is twice on the page.
  await expect(page.getByText('Entrada registrada').first()).toBeVisible();
}

/** Opens the cash at Principal with a float of 100.000; it goes straight to the POS. */
export async function openCash(page: Page) {
  await page.goto('/caja');
  await page.getByLabel('Base inicial').fill('100000');
  await page.getByRole('button', { name: 'Abrir caja' }).click();
  await expect(page).toHaveURL('/vender');
  await expect(page.getByText('Caja abierta en Principal')).toBeVisible();
}

/** Escapes text to match it literally in a RegExp. */
export function escape(text: string): string {
  return text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}
