import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type { components } from '@/shared/api/schema';
import { plain } from '@/test/fixtures';
import type { posApi } from './posApi';

export const user = () => userEvent.setup({ delay: null });

export async function searchBox() {
  return screen.findByRole('combobox', { name: 'Buscar producto' });
}

/** The POS focuses the search on the next animation frame: wait for it. */
export async function expectSearchFocused() {
  await waitFor(async () => expect(await searchBox()).toHaveFocus());
}

/** Types a code and Enter into the search, as a barcode reader does. */
export async function scan(code: string) {
  await user().type(await searchBox(), `${code}{Enter}`);
}

export function cartLine(name: RegExp | string) {
  return within(screen.getByRole('listitem', { name }));
}

export async function findCartLine(name: RegExp | string) {
  return within(await screen.findByRole('listitem', { name }));
}

/** An amount of the sale's summary by its term ("Total", "IVA"). */
export function summaryValue(term: string) {
  const summary = within(screen.getByRole('region', { name: 'Resumen de la venta' }));
  return plain(summary.getByText(term, { selector: 'dt' }).nextElementSibling?.textContent);
}

/** The body of the sale sent to the API. */
export async function posted(api: ReturnType<typeof posApi>) {
  const [request] = api.calls('POST /api/v1/sales');
  if (!request) throw new Error('El POS no envió la venta');
  return (await request.json()) as components['schemas']['ConfirmSaleInput'];
}

export const checkout = () => screen.findByRole('dialog', { name: 'Cobrar' });

/** The POS keeps the cart in this tab and the location and paper in the device: each test starts clean. */
export function clearPosStorage() {
  sessionStorage.clear();
  localStorage.clear();
}
