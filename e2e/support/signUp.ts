import { expect, type Page } from '@playwright/test';
import type { newBusiness } from './business';

/** Signs up the business with a template; the owner stays signed in, on the home page. */
export async function signUp(page: Page, business: ReturnType<typeof newBusiness>, template: string) {
  await page.goto('/registro');
  await page.getByLabel('Nombre de la empresa').fill(business.tenantName);
  await page.getByLabel('NIT', { exact: true }).fill(business.nit);
  await page.getByLabel('Su nombre').fill(business.ownerName);
  await page.getByLabel('Correo', { exact: true }).fill(business.email);
  await page.getByLabel('Contraseña', { exact: true }).fill(business.password);
  await page.getByRole('radio', { name: template }).check();
  await page.getByRole('button', { name: 'Crear cuenta' }).click();
  await expect(page).toHaveURL('/');
}

/** Creates the business's first location from the empty stock screen. */
export async function createLocation(page: Page, name: string) {
  await page.goto('/inventario');
  await page.getByRole('button', { name: 'Crear sede' }).click();
  await page.getByLabel('Nombre de la sede').fill(name);
  await page.getByRole('dialog').getByRole('button', { name: 'Crear sede' }).click();
  await expect(page.getByRole('dialog')).toBeHidden();
}

/** Picks an option of a Radix Select by the label of its field. */
export async function chooseOption(page: Page, label: string, option: RegExp) {
  await page.getByRole('combobox', { name: label }).click();
  await page.getByRole('option', { name: option }).click();
}
