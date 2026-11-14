import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { json } from '@/test/fakeApi';
import { renderApp } from '@/test/renderApp';
import { customersApi } from '../../testing/customersApi';
import { aBoutique, aBuilder, customerIds } from '@/test/fixtures';

async function choose(label: string | RegExp, option: string) {
  await userEvent.click(await screen.findByRole('combobox', { name: label }));
  await userEvent.click(await screen.findByRole('option', { name: option }));
}

describe('CustomerFormPage: datos de la factura electrónica', () => {
  it('elige el municipio del DANE por departamento y lo envía con su nombre como ciudad (ferretería)', async () => {
    const api = customersApi({ routes: { 'POST /api/v1/customers': json(201, aBuilder()) } });
    renderApp('/clientes/nuevo');

    await userEvent.type(await screen.findByLabelText('Número de documento'), '1020304050');
    await userEvent.type(screen.getByLabelText('Nombre'), 'Carlos Pérez');
    expect(screen.getByRole('combobox', { name: /^Municipio/ })).toBeDisabled();
    await choose(/^Departamento/, 'Antioquia');
    await choose(/^Municipio/, 'Envigado');
    await userEvent.click(screen.getByRole('button', { name: 'Guardar cliente' }));

    await waitFor(() => expect(api.calls('POST /api/v1/customers')).toHaveLength(1));
    expect(await api.calls('POST /api/v1/customers')[0]!.json()).toMatchObject({
      municipality_code: '05266',
      city: 'Envigado',
    });
  });

  it('muestra el municipio guardado del cliente (almacén de ropa en Medellín)', async () => {
    customersApi({ routes: { [`GET /api/v1/customers/${customerIds.boutique}`]: json(200, aBoutique()) } });
    renderApp(`/clientes/${customerIds.boutique}/editar`);

    expect(await screen.findByRole('combobox', { name: /^Departamento/ })).toHaveTextContent('Antioquia');
    expect(screen.getByRole('combobox', { name: /^Municipio/ })).toHaveTextContent('Medellín');
  });

  it('una empresa de otro país se identifica con su NIT extranjero', async () => {
    const api = customersApi({ routes: { 'POST /api/v1/customers': json(201, aBoutique()) } });
    renderApp('/clientes/nuevo');

    await choose('Tipo de persona', 'Persona jurídica');
    await choose('Tipo de documento', 'NIT de otro país');
    await userEvent.type(screen.getByLabelText('Número de documento'), 'B12345678');
    await userEvent.type(screen.getByLabelText('Razón social'), 'Textiles Ibéricos S.L.');
    await userEvent.click(screen.getByRole('button', { name: 'Guardar cliente' }));

    await waitFor(() => expect(api.calls('POST /api/v1/customers')).toHaveLength(1));
    expect(await api.calls('POST /api/v1/customers')[0]!.json()).toMatchObject({
      person_type: 'legal',
      document_type: 'foreign_nit',
    });
  });
});
