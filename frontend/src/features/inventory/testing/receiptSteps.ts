import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

export async function pickProduct(text: string, name: string) {
  await userEvent.type(await screen.findByRole('searchbox', { name: 'Producto' }), text);
  await userEvent.click(await screen.findByRole('button', { name: new RegExp(name) }));
}

export async function pickLocation(name: string) {
  await userEvent.click(screen.getByRole('combobox', { name: 'Sede' }));
  await userEvent.click(await screen.findByRole('option', { name }));
}

export const register = () => userEvent.click(screen.getByRole('button', { name: 'Registrar entrada' }));
