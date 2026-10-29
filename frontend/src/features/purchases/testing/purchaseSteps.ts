import { screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

/** Chooses an option of the select labelled label, inside container (the page by default). */
export async function choose(label: string, option: string, container = screen.getByRole('main')) {
  await userEvent.click(within(container).getByRole('combobox', { name: label }));
  await userEvent.click(await screen.findByRole('option', { name: option }));
}

/** Clears a field and types text in it. */
export async function retype(field: HTMLElement, text: string) {
  await userEvent.clear(field);
  await userEvent.type(field, text);
}

/** The group of the line n (from 1) of the purchase form. */
export const lineGroup = (n: number) => screen.getByRole('group', { name: `Línea ${String(n)}` });

/** Finds a product by text in line n and picks the option named name. */
export async function pickProduct(n: number, text: string, name: string) {
  await screen.findByRole('group', { name: `Línea ${String(n)}` });
  await userEvent.type(within(lineGroup(n)).getByRole('searchbox', { name: 'Producto' }), text);
  await userEvent.click(await within(lineGroup(n)).findByRole('button', { name: new RegExp(name) }));
}

/** Fills the invoice of the hardware distributor: location, number and date. */
export async function fillInvoice({ number = 'FE 1234', issuedOn = '2026-10-20' } = {}) {
  await screen.findByRole('combobox', { name: 'Proveedor' });
  await choose('Proveedor', 'Distribuidora Andina SAS');
  await choose('Sede', 'Bodega principal');
  await userEvent.type(screen.getByLabelText('Número de factura'), number);
  await retype(screen.getByLabelText('Fecha de la factura'), issuedOn);
}

/** Fills quantity and cost of line n. */
export async function fillAmounts(n: number, quantity: string, cost: string) {
  await userEvent.type(within(lineGroup(n)).getByLabelText('Cantidad'), quantity);
  await userEvent.type(within(lineGroup(n)).getByLabelText('Costo unitario'), cost);
}

export const registerPurchase = () => userEvent.click(screen.getByRole('button', { name: 'Registrar compra' }));
