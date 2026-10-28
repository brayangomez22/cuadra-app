import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

export function aFile(name = 'catalogo.xlsx') {
  return new File(['sku;nombre'], name);
}

export async function pickLocation(name: string) {
  await userEvent.click(await screen.findByRole('combobox', { name: 'Sede del stock inicial' }));
  await userEvent.click(await screen.findByRole('option', { name }));
}

// Without applyAccept, so a test can choose a file the picker would hide.
export async function chooseFile(file = aFile()) {
  await userEvent.setup({ applyAccept: false }).upload(await screen.findByLabelText('Archivo'), file);
}

export const validate = () => userEvent.click(screen.getByRole('button', { name: 'Revisar archivo' }));
export const confirm = () => userEvent.click(screen.getByRole('button', { name: 'Importar' }));

/**
 * The text fields of a multipart request, and its parts' names. Read from the
 * text: in the tests, Node cannot parse a FormData that carries a jsdom File,
 * and sends that file without its name or content (the E2E sends a real one).
 */
export async function sentFields(request: Request) {
  const fields: Record<string, string> = {};
  for (const part of (await request.text()).split(/\r\n--/)) {
    const name = /name="([^"]+)"/.exec(part)?.[1];
    if (name) fields[name] = part.split('\r\n\r\n')[1]?.trim() ?? '';
  }
  return fields;
}
