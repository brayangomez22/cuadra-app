import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser, json } from '@/test/fakeApi';
import { renderApp } from '@/test/renderApp';
import { generalApi } from '../../testing/settingsApi';

const owner = aUser('owner');

describe('GeneralSettings: analítica de uso', () => {
  it('una empresa nueva no comparte la analítica hasta que el dueño la acepta', async () => {
    // Una ferretería que acepta compartir cómo usa la app.
    let consent = false;
    const api = generalApi({
      'GET /api/v1/me': () => json(200, { ...owner, tenant_analytics_consent: consent }),
      'PATCH /api/v1/tenant': async (request) => {
        consent = ((await request.json()) as { analytics_consent: boolean }).analytics_consent;
        return json(200, {
          id: owner.tenant_id,
          name: owner.tenant_name,
          nit: owner.tenant_nit,
          business_template: owner.business_template,
          default_tax_rate: owner.default_tax_rate,
          address: '',
          city: '',
          phone: '',
          municipality_code: '',
          analytics_consent: consent,
        });
      },
    });
    renderApp('/configuracion/general');

    const box = await screen.findByRole('checkbox', { name: 'Compartir con Cuadra cómo se usa la app' });
    expect(box).not.toBeChecked();
    await userEvent.click(box);
    await userEvent.click(screen.getByRole('button', { name: 'Guardar preferencia' }));

    await waitFor(() => expect(api.calls('PATCH /api/v1/tenant')).toHaveLength(1));
    expect(await api.calls('PATCH /api/v1/tenant')[0]!.json()).toEqual({ analytics_consent: true });
    expect(await screen.findByText('Preferencia guardada')).toBeInTheDocument();
  });

  it('un almacén de calzado que ya aceptó la ve marcada', async () => {
    generalApi({}, { ...owner, business_template: 'ropa_calzado', tenant_analytics_consent: true });
    renderApp('/configuracion/general');

    expect(await screen.findByRole('checkbox', { name: 'Compartir con Cuadra cómo se usa la app' })).toBeChecked();
  });
});
