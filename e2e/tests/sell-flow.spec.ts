import { expect, test, type Page } from '@playwright/test';
import { newBusiness } from '../support/business';

// The critical flow of Cuadra, end to end: a business signs up, signs in again,
// creates a product, receives stock, sells with a mixed payment and finds the
// stock discounted. It runs once per business template; with ropa_calzado the
// product has variants (size × color) and the sale is of one of them.

type Flow = {
  template: string;
  product: { name: string; sku: string; price: string };
  /** The values to check in the variant matrix, and the variant that is received and sold. */
  variants?: { values: string[]; sold: { label: string; sku: string }; other: string };
  /** Price with VAT (19 %) × 3 units, split between cash and card. */
  total: string;
  cash: string;
  card: string;
};

const flows: Flow[] = [
  {
    template: 'Ferretería',
    product: { name: 'Martillo de uña 16 oz', sku: 'MART-16', price: '10000' },
    total: '35.700',
    cash: '20000',
    card: '15700',
  },
  {
    template: 'Ropa y calzado',
    product: { name: 'Camiseta básica', sku: 'CAMB', price: '20000' },
    variants: {
      values: ['M', 'L', 'Negro'],
      sold: { label: 'Negro · M', sku: 'CAMB-NEGRO-M' },
      other: 'Negro · L',
    },
    total: '71.400',
    cash: '50000',
    card: '21400',
  },
];

for (const flow of flows) {
  test.describe(`plantilla ${flow.template}`, () => {
    test('registra la empresa, ingresa de nuevo y vende con pago mixto descontando el stock', async ({ page }) => {
      const business = newBusiness(flow.template);
      const sold = flow.variants
        ? { name: `${flow.product.name} (${flow.variants.sold.label})`, sku: flow.variants.sold.sku }
        : { name: flow.product.name, sku: flow.product.sku };

      await test.step('registro', async () => {
        await page.goto('/registro');
        await page.getByLabel('Nombre de la empresa').fill(business.tenantName);
        await page.getByLabel('NIT', { exact: true }).fill(business.nit);
        await page.getByLabel('Su nombre').fill(business.ownerName);
        await page.getByLabel('Correo', { exact: true }).fill(business.email);
        await page.getByLabel('Contraseña', { exact: true }).fill(business.password);
        await page.getByRole('radio', { name: flow.template }).check();
        await page.getByRole('button', { name: 'Crear cuenta' }).click();
        await expect(page).toHaveURL('/');
        await page.getByRole('button', { name: 'Cerrar sesión' }).click();
        await expect(page.getByRole('heading', { name: 'Ingresar a Cuadra' })).toBeVisible();
      });

      await test.step('login', async () => {
        await page.getByLabel('Correo', { exact: true }).fill(business.email);
        await page.getByLabel('Contraseña', { exact: true }).fill(business.password);
        await page.getByRole('button', { name: 'Ingresar' }).click();
        await expect(page.getByRole('button', { name: 'Cerrar sesión' })).toBeVisible();
      });

      await test.step('crear producto', async () => {
        await page.goto('/productos/nuevo');
        await page.getByLabel('SKU', { exact: true }).fill(flow.product.sku);
        await page.getByLabel('Nombre', { exact: true }).fill(flow.product.name);
        await chooseOption(page, 'Unidad', /^Unidad \(und\)$/);
        await page.getByLabel('Costo', { exact: true }).fill('6000');
        await page.getByLabel('Precio de venta (sin IVA)').fill(flow.product.price);
        for (const value of flow.variants?.values ?? []) {
          await page.getByRole('checkbox', { name: value, exact: true }).check();
        }
        if (flow.variants) {
          await expect(page.getByRole('table', { name: 'Variantes del producto' }).getByRole('row')).toHaveCount(3);
        }
        await page.getByRole('button', { name: 'Guardar producto' }).click();
        await expect(page).toHaveURL('/productos');
        await expect(page.getByText(flow.product.name).first()).toBeVisible();
      });

      await test.step('registrar entrada de inventario', async () => {
        await page.goto('/inventario');
        await page.getByRole('button', { name: 'Crear sede' }).click();
        await page.getByLabel('Nombre de la sede').fill('Principal');
        await page.getByRole('dialog').getByRole('button', { name: 'Crear sede' }).click();
        await page.getByRole('link', { name: 'Registrar entrada' }).first().click();
        await page.getByLabel('Producto', { exact: true }).fill(sold.sku);
        await page.getByRole('list', { name: 'Resultados' }).getByRole('button', { name: sold.name }).click();
        await page.getByLabel('Cantidad', { exact: true }).fill('10');
        await page.getByRole('button', { name: 'Registrar entrada' }).click();
        // The toast is announced too: the same text is twice on the page.
        await expect(page.getByText('Entrada registrada').first()).toBeVisible();
      });

      await test.step('abrir la caja', async () => {
        await page.goto('/caja');
        await page.getByLabel('Base inicial').fill('100000');
        await page.getByRole('button', { name: 'Abrir caja' }).click();
        // Opening the cash goes straight to the POS.
        await expect(page).toHaveURL('/vender');
        await expect(page.getByText('Caja abierta en Principal')).toBeVisible();
      });

      await test.step('vender con pago mixto', async () => {
        await page.goto('/vender');
        const search = page.getByRole('combobox', { name: 'Buscar producto' });
        await search.fill(sold.sku);
        await page.getByRole('option', { name: new RegExp(escape(sold.name)) }).click();
        await page.getByLabel(/^Cantidad/).fill('3');
        const summary = page.getByRole('region', { name: 'Resumen de la venta' });
        await expect(summary).toContainText(flow.total);
        await page.getByRole('button', { name: /^Cobrar/ }).click();

        const checkout = page.getByRole('dialog', { name: 'Cobrar' });
        await checkout.getByLabel('Valor del pago 1').fill(flow.cash);
        await checkout.getByRole('button', { name: 'Agregar medio de pago' }).click();
        await checkout.getByLabel('Valor del pago 2').fill(flow.card);
        await checkout.getByRole('button', { name: 'Confirmar venta' }).click();

        const result = page.getByRole('dialog', { name: /^Venta N\.º/ });
        const payments = result.getByRole('list', { name: 'Pagos de la venta' });
        await expect(payments.getByRole('listitem')).toHaveCount(2);
        await expect(payments).toContainText('Efectivo');
        await expect(payments).toContainText('Tarjeta');
        await expect(result).toContainText(flow.total);
        await result.getByRole('button', { name: 'Nueva venta' }).click();
      });

      await test.step('verificar el stock', async () => {
        await page.goto('/inventario');
        const stock = page.getByRole('table', { name: 'Existencias en Principal' });
        // 10 received − 3 sold. The row name also carries the SKU and the figures, hence the regex.
        const row = stock.getByRole('row', { name: new RegExp(`^${escape(sold.name)}`) });
        await expect(row.getByRole('cell').nth(1)).toHaveText('7 und');
        if (flow.variants) {
          // Only the variant sold moves: the other one never had stock.
          const other = new RegExp(`^${escape(`${flow.product.name} (${flow.variants.other})`)}`);
          await expect(stock.getByRole('row', { name: other })).toHaveCount(0);
        }
      });
    });
  });
}

/** Picks an option of a Radix Select by the label of its field. */
async function chooseOption(page: Page, label: string, option: RegExp) {
  await page.getByRole('combobox', { name: label }).click();
  await page.getByRole('option', { name: option }).click();
}

function escape(text: string): string {
  return text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}
