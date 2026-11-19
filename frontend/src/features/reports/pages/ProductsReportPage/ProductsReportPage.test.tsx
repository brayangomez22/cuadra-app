import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { json } from '@/test/fakeApi';
import { renderApp } from '@/test/renderApp';
import { productsReport } from '../../testing/productFixtures';
import { reportsApi } from '../../testing/reportsApi';

const productsKey = 'GET /api/v1/reports/products';
const url = '/reportes/productos?desde=2026-11-01&hasta=2026-11-19';

describe('ProductsReportPage', () => {
  it('muestra cada producto con sus tallas debajo y lo que dejó', async () => {
    reportsApi({ [productsKey]: json(200, productsReport) });
    renderApp(url);

    const shirt = within((await screen.findByRole('cell', { name: /^Camiseta básica/ })).closest('tr')!);
    expect(shirt.getByText('$ 210.000')).toBeInTheDocument();
    expect(shirt.getByText('60,0 %')).toBeInTheDocument();
    expect(screen.getByRole('cell', { name: 'Rojo · L' })).toBeInTheDocument();
    expect(screen.getByRole('cell', { name: /^Cemento gris 50kg/ })).toBeInTheDocument();
  });

  it('filtra por un atributo de lista de la empresa con sus opciones', async () => {
    const api = reportsApi({ [productsKey]: json(200, productsReport) });
    renderApp(url);

    await userEvent.click(await screen.findByRole('combobox', { name: 'Atributo' }));
    await userEvent.click(await screen.findByRole('option', { name: 'Material' }));
    await userEvent.click(await screen.findByRole('combobox', { name: 'Material' }));
    await userEvent.click(await screen.findByRole('option', { name: 'Algodón' }));

    await waitFor(() => {
      const asked = new URL(api.calls(productsKey).at(-1)!.url);
      expect(asked.searchParams.get('attribute')).toBe('material');
      expect(asked.searchParams.get('attribute_value')).toBe('Algodón');
    });
  });

  it('ordena por utilidad', async () => {
    const api = reportsApi({ [productsKey]: json(200, productsReport) });
    renderApp(url);

    await userEvent.click(await screen.findByRole('combobox', { name: 'Ordenar por' }));
    await userEvent.click(await screen.findByRole('option', { name: 'Más utilidad' }));

    await waitFor(() => {
      expect(new URL(api.calls(productsKey).at(-1)!.url).searchParams.get('sort')).toBe('profit');
    });
  });
});
