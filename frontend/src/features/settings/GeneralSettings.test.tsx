import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser, fakeApi, json } from '@/test/fakeApi';
import { renderApp } from '@/test/renderApp';

describe('GeneralSettings', () => {
  it('muestra el IVA por defecto de la empresa', async () => {
    // Droguería: medicamentos excluidos, 0 %.
    fakeApi({
      'GET /api/v1/me': json(200, { ...aUser('owner'), business_template: 'drogueria', default_tax_rate: '0' }),
    });
    renderApp('/configuracion/general');

    expect(await screen.findByRole('combobox', { name: 'IVA por defecto' })).toHaveTextContent(/^0\s%$/);
  });

  it('guarda el IVA por defecto con PATCH /tenant y /me lo refleja', async () => {
    // Ferretería que pasa a proponer 5 %.
    let rate = '0.19';
    const api = fakeApi({
      'GET /api/v1/me': () => json(200, { ...aUser('owner'), default_tax_rate: rate }),
      'PATCH /api/v1/tenant': async (request) => {
        rate = ((await request.json()) as { default_tax_rate: string }).default_tax_rate;
        return json(200, {
          id: aUser().tenant_id,
          name: 'Ferretería El Tornillo',
          nit: '900123456-8',
          business_template: 'ferreteria',
          default_tax_rate: rate,
        });
      },
    });
    renderApp('/configuracion/general');

    await userEvent.click(await screen.findByRole('combobox', { name: 'IVA por defecto' }));
    await userEvent.click(await screen.findByRole('option', { name: /^5\s%$/ }));
    await userEvent.click(screen.getByRole('button', { name: 'Guardar' }));

    expect(await screen.findByText('Configuración guardada')).toBeInTheDocument();
    expect(await api.calls('PATCH /api/v1/tenant')[0]!.json()).toEqual({ default_tax_rate: '0.05' });
    await waitFor(() => expect(api.calls('GET /api/v1/me').length).toBeGreaterThanOrEqual(2));
  });

  it('el administrador no ve la configuración de la empresa', async () => {
    fakeApi({ 'GET /api/v1/me': json(200, aUser('admin')) });
    renderApp('/configuracion/general');

    expect(await screen.findByText('No tiene permiso para ver esta sección')).toBeInTheDocument();
  });
});
