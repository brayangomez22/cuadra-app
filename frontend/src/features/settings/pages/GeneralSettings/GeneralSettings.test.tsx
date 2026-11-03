import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser, json } from '@/test/fakeApi';
import { renderApp } from '@/test/renderApp';
import { generalApi } from '../../testing/settingsApi';

const owner = aUser('owner');

// tenant is the answer of PATCH /tenant: the business with the given changes.
function tenant(changes: Record<string, string>) {
  return {
    id: owner.tenant_id,
    name: owner.tenant_name,
    nit: owner.tenant_nit,
    business_template: owner.business_template,
    default_tax_rate: owner.default_tax_rate,
    address: '',
    city: '',
    phone: '',
    ...changes,
  };
}

describe('GeneralSettings', () => {
  it('muestra el IVA por defecto de la empresa', async () => {
    // Droguería: medicamentos excluidos, 0 %.
    generalApi({}, { ...owner, business_template: 'drogueria', default_tax_rate: '0' });
    renderApp('/configuracion/general');

    expect(await screen.findByRole('combobox', { name: 'IVA por defecto' })).toHaveTextContent(/^0\s%$/);
  });

  it('guarda el IVA por defecto con PATCH /tenant y /me lo refleja', async () => {
    // Ferretería que pasa a proponer 5 %.
    let rate = '0.19';
    const api = generalApi({
      'GET /api/v1/me': () => json(200, { ...owner, default_tax_rate: rate }),
      'PATCH /api/v1/tenant': async (request) => {
        rate = ((await request.json()) as { default_tax_rate: string }).default_tax_rate;
        return json(200, tenant({ default_tax_rate: rate }));
      },
    });
    renderApp('/configuracion/general');

    await userEvent.click(await screen.findByRole('combobox', { name: 'IVA por defecto' }));
    await userEvent.click(await screen.findByRole('option', { name: /^5\s%$/ }));
    await userEvent.click(screen.getByRole('button', { name: 'Guardar IVA' }));

    expect(await screen.findByText('IVA guardado')).toBeInTheDocument();
    expect(await api.calls('PATCH /api/v1/tenant')[0]!.json()).toEqual({ default_tax_rate: '0.05' });
    await waitFor(() => expect(api.calls('GET /api/v1/me').length).toBeGreaterThanOrEqual(2));
  });

  it('muestra y guarda la dirección, la ciudad y el teléfono del negocio', async () => {
    // Almacén de ropa que registra sus datos para la tirilla.
    const api = generalApi(
      {
        'PATCH /api/v1/tenant': json(
          200,
          tenant({ address: 'Cra. 7 # 20-15', city: 'Bogotá D.C.', phone: '601 300 1234' }),
        ),
      },
      { ...owner, business_template: 'ropa_calzado', tenant_city: 'Bogotá D.C.' },
    );
    renderApp('/configuracion/general');

    expect(await screen.findByLabelText('Ciudad o municipio (opcional)')).toHaveValue('Bogotá D.C.');
    await userEvent.type(screen.getByLabelText('Dirección (opcional)'), ' Cra. 7 # 20-15 ');
    await userEvent.type(screen.getByLabelText('Teléfono (opcional)'), '601 300 1234');
    await userEvent.click(screen.getByRole('button', { name: 'Guardar datos del negocio' }));

    expect(await screen.findByText('Datos del negocio guardados')).toBeInTheDocument();
    expect(await api.calls('PATCH /api/v1/tenant')[0]!.json()).toEqual({
      address: 'Cra. 7 # 20-15',
      city: 'Bogotá D.C.',
      phone: '601 300 1234',
    });
  });

  it('marca el teléfono que la API rechaza', async () => {
    generalApi({
      'PATCH /api/v1/tenant': json(422, {
        error: { code: 'invalid_tenant_phone', message: 'Escriba el teléfono solo con números.' },
      }),
    });
    renderApp('/configuracion/general');

    await userEvent.type(await screen.findByLabelText('Teléfono (opcional)'), '300 123 4567');
    await userEvent.click(screen.getByRole('button', { name: 'Guardar datos del negocio' }));

    expect(await screen.findByLabelText('Teléfono (opcional)')).toHaveAccessibleDescription(
      'Escriba el teléfono solo con números.',
    );
  });

  it('no envía un teléfono con letras', async () => {
    const api = generalApi();
    renderApp('/configuracion/general');

    await userEvent.type(await screen.findByLabelText('Teléfono (opcional)'), 'llamar al dueño');
    await userEvent.click(screen.getByRole('button', { name: 'Guardar datos del negocio' }));

    expect(
      await screen.findByText('Escriba el teléfono solo con números, espacios, +, - o paréntesis'),
    ).toBeInTheDocument();
    expect(api.calls('PATCH /api/v1/tenant')).toHaveLength(0);
  });

  it('el administrador no ve la configuración de la empresa', async () => {
    generalApi({}, aUser('admin'));
    renderApp('/configuracion/general');

    expect(await screen.findByText('No tiene permiso para ver esta sección')).toBeInTheDocument();
  });
});
