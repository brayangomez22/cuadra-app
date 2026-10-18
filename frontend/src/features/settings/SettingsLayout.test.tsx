import { screen, waitFor, within } from '@testing-library/react';
import { aUser, fakeApi, json, signedIn, type Me } from '@/test/fakeApi';
import { attributes, axes, page, units } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';

function settingsApi(user: Me) {
  return fakeApi({
    ...signedIn(user),
    'GET /api/v1/units': json(200, page(units)),
    'GET /api/v1/product-attributes': json(200, page(attributes)),
    'GET /api/v1/variant-axes': json(200, page(axes)),
    'GET /api/v1/modules': json(200, { items: [] }),
  });
}

async function sections() {
  const nav = await screen.findByRole('navigation', { name: 'Secciones de configuración' });
  return within(nav)
    .getAllByRole('link')
    .map((link) => link.textContent);
}

describe('SettingsLayout', () => {
  it('al dueño le muestra todas las secciones; los ejes, solo con el módulo variants', async () => {
    settingsApi(aUser('owner'));
    renderApp('/configuracion/unidades');
    expect(await sections()).toEqual(['General', 'Unidades', 'Atributos', 'Módulos']);
  });

  it('con el módulo variants activo aparece la sección de ejes de variante', async () => {
    settingsApi({ ...aUser('owner'), business_template: 'ropa_calzado', modules: ['variants'] });
    renderApp('/configuracion/unidades');
    expect(await sections()).toEqual(['General', 'Unidades', 'Atributos', 'Ejes de variante', 'Módulos']);
  });

  it('al administrador no le muestra la configuración de la empresa ni los módulos', async () => {
    settingsApi(aUser('admin'));
    renderApp('/configuracion/unidades');
    expect(await sections()).toEqual(['Unidades', 'Atributos']);
  });

  it('abre la primera sección que el rol puede ver', async () => {
    settingsApi(aUser('admin'));
    const { router } = renderApp('/configuracion');
    await waitFor(() => expect(router.state.location.pathname).toBe('/configuracion/unidades'));
  });

  it('la sección de ejes con el módulo inactivo responde que no existe', async () => {
    settingsApi(aUser('owner'));
    renderApp('/configuracion/ejes');
    expect(await screen.findByText('Esta sección no está activa')).toBeInTheDocument();
  });

  it('el cajero no tiene acceso a la configuración', async () => {
    settingsApi(aUser('cashier'));
    renderApp('/configuracion/unidades');
    expect(await screen.findByText('No tiene permiso para ver esta sección')).toBeInTheDocument();
  });
});
