import { cleanup, screen } from '@testing-library/react';
import { renderApp } from '@/test/renderApp';
import { preloadPages } from '@/test/preloadPages';
import { purchasesApi } from './purchasesApi';
import { choose } from './purchaseSteps';

/**
 * For a beforeAll of the purchase form's tests: loads the screen and renders
 * it once, opening a select, then leaves nothing behind. In a cold worker
 * that first render and the first select take about a second, which the
 * first test would pay against its timeout (P146).
 */
export function warmUpPurchaseForm() {
  return async () => {
    await preloadPages('/compras/nueva')();
    purchasesApi();
    renderApp('/compras/nueva');
    await screen.findByRole('combobox', { name: 'Proveedor' });
    await choose('Proveedor', 'Distribuidora Andina SAS');
    cleanup();
    vi.unstubAllGlobals();
  };
}
