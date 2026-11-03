import { expect, type Page } from '@playwright/test';
import { escape } from './stock';

/**
 * Sells quantity of the variant in the POS, paid in cash with exactly total
 * (with VAT, "23.800"), and leaves the POS ready for the next sale. Returns
 * the sale's number ("N.º 1").
 */
export async function sellInCash(page: Page, variant: { name: string; sku: string }, quantity: string, total: string) {
  await page.goto('/vender');
  await page.getByRole('combobox', { name: 'Buscar producto' }).fill(variant.sku);
  await page.getByRole('option', { name: new RegExp(escape(variant.name)) }).click();
  await page.getByLabel(/^Cantidad/).fill(quantity);
  await expect(page.getByRole('region', { name: 'Resumen de la venta' })).toContainText(total);
  await page.getByRole('button', { name: /^Cobrar/ }).click();

  const checkout = page.getByRole('dialog', { name: 'Cobrar' });
  await checkout.getByLabel('Valor del pago 1').fill(total.replaceAll('.', ''));
  await checkout.getByRole('button', { name: 'Confirmar venta' }).click();

  const result = page.getByRole('dialog', { name: /^Venta N\.º/ });
  await expect(result).toContainText(total);
  const number = /N\.º \d+/.exec((await result.getByRole('heading').first().textContent()) ?? '')?.[0] ?? '';
  await result.getByRole('button', { name: 'Nueva venta' }).click();
  return number;
}
