import { screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderApp } from '@/test/renderApp';
import { preloadPages } from '@/test/preloadPages';
import { purchasesApi } from '../../testing/purchasesApi';
import { lineGroup } from '../../testing/purchaseSteps';

describe('PurchaseFormPage: teclado', () => {
  beforeAll(preloadPages('/compras/nueva'));

  it('las flechas recorren los resultados del producto y Enter elige uno', async () => {
    purchasesApi();
    renderApp('/compras/nueva');

    await screen.findByRole('group', { name: 'Línea 1' });
    const search = within(lineGroup(1)).getByRole('searchbox', { name: 'Producto' });
    await userEvent.type(search, 'c');
    const tube = await within(lineGroup(1)).findByRole('button', { name: /Tubo PVC/ });

    await userEvent.keyboard('{ArrowDown}');
    expect(tube).toHaveFocus();
    await userEvent.keyboard('{ArrowDown}');
    expect(within(lineGroup(1)).getByRole('button', { name: /Cemento gris/ })).toHaveFocus();
    await userEvent.keyboard('{ArrowUp}{ArrowUp}');
    expect(search).toHaveFocus();
    await userEvent.keyboard('{ArrowDown}{ArrowDown}{Enter}');

    expect(await within(lineGroup(1)).findByText(/Cemento gris 50kg/)).toBeInTheDocument();
    expect(within(lineGroup(1)).queryByRole('searchbox', { name: 'Producto' })).not.toBeInTheDocument();
  });

  it('«Agregar línea» lleva al producto de la línea nueva', async () => {
    purchasesApi();
    renderApp('/compras/nueva');

    await screen.findByRole('group', { name: 'Línea 1' });
    await userEvent.click(screen.getByRole('button', { name: 'Agregar línea' }));

    expect(within(lineGroup(2)).getByRole('searchbox', { name: 'Producto' })).toHaveFocus();
  });
});
