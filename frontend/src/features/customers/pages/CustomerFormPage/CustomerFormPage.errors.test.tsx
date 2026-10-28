import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { apiError } from '@/test/fakeApi';
import { renderApp } from '@/test/renderApp';
import { customersApi } from '../../testing/customersApi';

async function submitNit(number: string) {
  await userEvent.click(await screen.findByRole('combobox', { name: 'Tipo de documento' }));
  await userEvent.click(await screen.findByRole('option', { name: 'NIT' }));
  await userEvent.type(screen.getByLabelText('Número de documento'), number);
  await userEvent.type(screen.getByLabelText('Nombre'), 'Ana Ruiz');
  await userEvent.click(screen.getByRole('button', { name: 'Guardar cliente' }));
}

describe('CustomerFormPage: errores', () => {
  it('muestra el error del DV junto al número', async () => {
    customersApi({
      routes: {
        'POST /api/v1/customers': apiError(422, {
          code: 'invalid_nit_check_digit',
          message: 'El dígito de verificación del NIT no coincide.',
        }),
      },
    });
    renderApp('/clientes/nuevo');

    await submitNit('901234567-1');

    const number = screen.getByLabelText('Número de documento');
    expect(await screen.findByText('El dígito de verificación del NIT no coincide.')).toBeInTheDocument();
    expect(number).toHaveAttribute('aria-invalid', 'true');
    expect(number).toHaveFocus();
  });

  it('muestra el documento repetido junto al número', async () => {
    customersApi({
      routes: {
        'POST /api/v1/customers': apiError(409, {
          code: 'customer_document_taken',
          message: 'Ya existe un cliente con el NIT 901234567-7.',
        }),
      },
    });
    renderApp('/clientes/nuevo');

    await submitNit('901234567-7');

    expect(await screen.findByText('Ya existe un cliente con el NIT 901234567-7.')).toBeInTheDocument();
    expect(screen.getByLabelText('Número de documento')).toHaveAttribute('aria-invalid', 'true');
  });
});
