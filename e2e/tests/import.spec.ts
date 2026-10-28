import { expect, test } from '@playwright/test';
import { newBusiness } from '../support/business';
import { createLocation, signUp } from '../support/signUp';

// The initial import, end to end: a business downloads the template, uploads
// a file with an error and sees it by row, fixes it and imports it, and finds
// its products in the catalog. It runs once per business template; with
// ropa_calzado the file has a product with two variants (size × color).

type Flow = {
  template: string;
  /** A .csv separated by ";" as Excel saves it in Colombia, with the template's columns. */
  header: string;
  rows: string[];
  /** The row with an error and the message the API answers. */
  wrong: { row: string; message: string };
  products: string[];
  created: string;
};

const flows: Flow[] = [
  {
    template: 'Ferretería',
    header: 'producto;unidad;sku;costo;precio;stock_inicial',
    rows: ['Martillo de uña 16 oz;und;MART-16;12000;25000;12', 'Tornillo drywall 6x1;und;TORN-6;50;150,50;500'],
    wrong: {
      row: 'Martillo de uña 16 oz;und;MART-16;12000;veinte mil;12',
      message: 'Fila 2, columna «precio»: no es un número.',
    },
    products: ['Martillo de uña 16 oz', 'Tornillo drywall 6x1'],
    created: '2 creados',
  },
  {
    template: 'Ropa y calzado',
    header: 'producto;unidad;sku;costo;precio;stock_inicial;atributo: Género;eje: talla;eje: color',
    rows: [
      'Camiseta básica;und;CAMB-S-NEGRO;15000;30000;10;Unisex;S;Negro',
      'Camiseta básica;;CAMB-M-NEGRO;15000;30000;8;;M;Negro',
      'Correa de cuero;und;COR-1;20000;45000;;Unisex;;',
    ],
    wrong: { row: 'Camiseta básica;und;CAMB-S-NEGRO;15000;30000;10;Unisex;S;Violeta', message: 'Fila 2' },
    products: ['Camiseta básica', 'Correa de cuero'],
    created: '2 creados',
  },
];

const csv = (header: string, rows: string[]) => ({
  name: 'productos.csv',
  mimeType: 'text/csv',
  buffer: Buffer.from([header, ...rows].join('\r\n') + '\r\n'),
});

for (const flow of flows) {
  test.describe(`plantilla ${flow.template}`, () => {
    test('importa un archivo pequeño después de corregir su error', async ({ page }) => {
      await signUp(page, newBusiness(flow.template), flow.template);
      await createLocation(page, 'Principal');

      await test.step('descargar la plantilla', async () => {
        await page.goto('/productos');
        await page.getByRole('link', { name: 'Importar desde Excel' }).click();
        const download = page.waitForEvent('download');
        await page.getByRole('button', { name: 'Plantilla .xlsx' }).click();
        expect((await download).suggestedFilename()).toBe('plantilla-productos.xlsx');
      });

      await test.step('un archivo con un error muestra su fila y no se puede importar', async () => {
        await page.getByLabel('Archivo', { exact: true }).setInputFiles(csv(flow.header, [flow.wrong.row]));
        await page.getByRole('button', { name: 'Revisar archivo' }).click();
        const errors = page.getByRole('table', { name: 'Errores del archivo' });
        await expect(errors).toContainText(flow.wrong.message);
        await expect(page.getByRole('button', { name: 'Importar' })).toHaveCount(0);
        await page.getByRole('button', { name: 'Subir otro archivo' }).click();
      });

      await test.step('el archivo corregido se importa', async () => {
        await page.getByLabel('Archivo', { exact: true }).setInputFiles(csv(flow.header, flow.rows));
        await page.getByRole('button', { name: 'Revisar archivo' }).click();
        await expect(page.getByRole('table', { name: 'Primeros productos del archivo' })).toContainText(
          flow.products[0],
        );
        await page.getByRole('button', { name: 'Importar' }).click();
        await expect(page.getByRole('heading', { name: 'Importación terminada' })).toBeVisible({ timeout: 30_000 });
        await expect(page.getByRole('list', { name: 'Resultado de la importación' })).toContainText(flow.created);
      });

      await test.step('los productos quedan en el catálogo', async () => {
        await page.getByRole('link', { name: 'Ver productos' }).click();
        const products = page.getByRole('table', { name: 'Productos' });
        for (const name of flow.products) await expect(products).toContainText(name);
      });
    });
  });
}
