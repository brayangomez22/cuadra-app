import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser, fakeApi, json } from '@/test/fakeApi';
import { renderApp } from '@/test/renderApp';
import { clothingId, pilotId, someFlags } from '../../testing/platformApi';

const operator = { ...aUser('owner'), operator: true };
const noContent = () => new Response(null, { status: 204 });

function card(key: string) {
  return screen.getByRole('region', { name: key });
}

describe('FlagsPage', () => {
  it('muestra cada flag con su valor global y sus excepciones', async () => {
    fakeApi({ 'GET /api/v1/me': json(200, operator), 'GET /api/v1/platform/flags': json(200, { items: someFlags() }) });
    renderApp('/interno/flags');

    expect(await screen.findByRole('region', { name: 'feedback' })).toBeInTheDocument();
    expect(
      within(card('feedback')).getByRole('checkbox', { name: 'Prendida para todas las empresas' }),
    ).not.toBeChecked();
    expect(within(card('feedback')).getByText('Ferretería La Tuerca')).toBeInTheDocument();
    expect(within(card('analytics')).getByRole('checkbox', { name: 'Prendida para todas las empresas' })).toBeChecked();
    expect(within(card('analytics')).getByText('Ninguna empresa tiene una excepción.')).toBeInTheDocument();
  });

  it('prende una flag para todas las empresas', async () => {
    const api = fakeApi({
      'GET /api/v1/me': json(200, operator),
      'GET /api/v1/platform/flags': json(200, { items: someFlags() }),
      'PUT /api/v1/platform/flags/feedback': json(200, { key: 'feedback', description: '', enabled: true }),
    });
    renderApp('/interno/flags');

    await userEvent.click(
      within(await screen.findByRole('region', { name: 'feedback' })).getByRole('checkbox', {
        name: 'Prendida para todas las empresas',
      }),
    );

    await waitFor(() => expect(api.calls('PUT /api/v1/platform/flags/feedback')).toHaveLength(1));
    expect(await api.calls('PUT /api/v1/platform/flags/feedback')[0]!.json()).toEqual({ enabled: true });
  });

  it('apaga la flag para una empresa con una excepción', async () => {
    // Un almacén de calzado al que algo le falla con la analítica.
    const path = `PUT /api/v1/platform/flags/analytics/tenants/${clothingId}`;
    const api = fakeApi({
      'GET /api/v1/me': json(200, operator),
      'GET /api/v1/platform/flags': json(200, { items: someFlags() }),
      [path]: noContent,
    });
    renderApp('/interno/flags');
    const analytics = await screen.findByRole('region', { name: 'analytics' });

    await userEvent.type(within(analytics).getByLabelText('ID de la empresa'), clothingId);
    await userEvent.click(within(analytics).getByRole('radio', { name: 'Apagada' }));
    await userEvent.click(within(analytics).getByRole('button', { name: 'Agregar excepción' }));

    await waitFor(() => expect(api.calls(path)).toHaveLength(1));
    expect(await api.calls(path)[0]!.json()).toEqual({ enabled: false });
  });

  it('no envía una excepción sin un ID de empresa válido', async () => {
    const api = fakeApi({
      'GET /api/v1/me': json(200, operator),
      'GET /api/v1/platform/flags': json(200, { items: someFlags() }),
    });
    renderApp('/interno/flags');
    const analytics = await screen.findByRole('region', { name: 'analytics' });

    await userEvent.type(within(analytics).getByLabelText('ID de la empresa'), 'Calzado Andino');
    await userEvent.click(within(analytics).getByRole('button', { name: 'Agregar excepción' }));

    expect(await within(analytics).findByText(/Escriba el ID de la empresa/)).toBeInTheDocument();
    expect(api.calls(`PUT /api/v1/platform/flags/analytics/tenants/${clothingId}`)).toHaveLength(0);
  });

  it('quita la excepción de una empresa', async () => {
    const path = `DELETE /api/v1/platform/flags/feedback/tenants/${pilotId}`;
    const api = fakeApi({
      'GET /api/v1/me': json(200, operator),
      'GET /api/v1/platform/flags': json(200, { items: someFlags() }),
      [path]: noContent,
    });
    renderApp('/interno/flags');

    await userEvent.click(
      within(await screen.findByRole('region', { name: 'feedback' })).getByRole('button', {
        name: 'Quitar la excepción de Ferretería La Tuerca',
      }),
    );

    await waitFor(() => expect(api.calls(path)).toHaveLength(1));
  });

  it('para quien no es operador la página no existe', async () => {
    const api = fakeApi({ 'GET /api/v1/me': json(200, aUser('owner')) });
    renderApp('/interno/flags');

    expect(await screen.findByText('Esta página no existe')).toBeInTheDocument();
    expect(api.calls('GET /api/v1/platform/flags')).toHaveLength(0);
  });
});
