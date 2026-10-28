import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { json } from '@/test/fakeApi';
import { renderApp } from '@/test/renderApp';
import { customersApi } from '../../testing/customersApi';
import { aBoutique, aBuilder, segmentIds } from '@/test/fixtures';

async function choose(label: string | RegExp, option: string) {
  await userEvent.click(await screen.findByRole('combobox', { name: label }));
  await userEvent.click(await screen.findByRole('option', { name: option }));
}

describe('CustomerFormPage: crear', () => {
  it('crea una persona natural y vuelve a la lista', async () => {
    const api = customersApi({ routes: { 'POST /api/v1/customers': json(201, aBuilder()) } });
    renderApp('/clientes/nuevo');

    await userEvent.type(await screen.findByLabelText('Número de documento'), '1020304050');
    await userEvent.type(screen.getByLabelText('Nombre'), 'Carlos Pérez');
    await userEvent.type(screen.getByLabelText('Teléfono (opcional)'), '3001234567');
    await choose(/^Segmento/, 'Maestro de obra');
    await userEvent.click(screen.getByRole('button', { name: 'Guardar cliente' }));

    await waitFor(() => expect(api.calls('POST /api/v1/customers')).toHaveLength(1));
    expect(await api.calls('POST /api/v1/customers')[0]!.json()).toEqual({
      person_type: 'natural',
      document_type: 'cc',
      document_number: '1020304050',
      name: 'Carlos Pérez',
      phone: '3001234567',
      segment_id: segmentIds.builders,
      tax_responsibilities: ['R-99-PN'],
      active: true,
    });
    expect(await screen.findByRole('heading', { name: 'Clientes' })).toBeInTheDocument();
  });

  it('una persona jurídica solo ofrece NIT', async () => {
    const api = customersApi({ routes: { 'POST /api/v1/customers': json(201, aBoutique()) } });
    renderApp('/clientes/nuevo');

    await choose('Tipo de persona', 'Persona jurídica');
    expect(screen.getByRole('combobox', { name: 'Tipo de documento' })).toHaveTextContent('NIT');
    await userEvent.click(screen.getByRole('combobox', { name: 'Tipo de documento' }));
    expect(await screen.findAllByRole('option')).toHaveLength(1);
    await userEvent.keyboard('{Escape}');

    await userEvent.type(screen.getByLabelText('Número de documento'), '901234567-7');
    await userEvent.type(screen.getByLabelText('Razón social'), 'Confecciones Sol SAS');
    await userEvent.click(screen.getByRole('checkbox', { name: /Gran contribuyente/ }));
    await userEvent.click(screen.getByRole('button', { name: 'Guardar cliente' }));

    await waitFor(() => expect(api.calls('POST /api/v1/customers')).toHaveLength(1));
    expect(await api.calls('POST /api/v1/customers')[0]!.json()).toMatchObject({
      person_type: 'legal',
      document_type: 'nit',
      tax_responsibilities: ['O-13'],
    });
  });
});
