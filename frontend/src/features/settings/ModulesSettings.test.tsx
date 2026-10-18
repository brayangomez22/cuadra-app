import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser, apiError, fakeApi, json } from '@/test/fakeApi';
import { renderApp } from '@/test/renderApp';

const variants = { id: 'variants', name: 'Variantes', depends_on: [], active: false };

describe('ModulesSettings', () => {
  it('lista los módulos con su estado', async () => {
    fakeApi({
      'GET /api/v1/me': json(200, aUser('owner')),
      'GET /api/v1/modules': json(200, { items: [variants] }),
    });
    renderApp('/configuracion/modulos');

    expect(await screen.findByRole('checkbox', { name: 'Variantes' })).not.toBeChecked();
  });

  it('activa un módulo y vuelve a pedir /me para mostrar sus menús', async () => {
    // Una ferretería que empieza a vender ropa de trabajo por talla.
    let active = false;
    const api = fakeApi({
      'GET /api/v1/me': () => json(200, { ...aUser('owner'), modules: active ? ['variants'] : [] }),
      'GET /api/v1/modules': () => json(200, { items: [{ ...variants, active }] }),
      'PUT /api/v1/modules/variants': () => {
        active = true;
        return json(200, { ...variants, active: true });
      },
    });
    renderApp('/configuracion/modulos');

    await userEvent.click(await screen.findByRole('checkbox', { name: 'Variantes' }));

    await waitFor(() => expect(screen.getByRole('checkbox', { name: 'Variantes' })).toBeChecked());
    expect(await api.calls('PUT /api/v1/modules/variants')[0]!.json()).toEqual({ active: true });
    expect(await screen.findByRole('link', { name: 'Ejes de variante' })).toBeInTheDocument();
    expect(api.calls('GET /api/v1/me').length).toBeGreaterThanOrEqual(2);
  });

  it('muestra el módulo que bloquea el cambio y deja el estado como estaba', async () => {
    fakeApi({
      'GET /api/v1/me': json(200, { ...aUser('owner'), modules: ['variants'] }),
      'GET /api/v1/modules': json(200, { items: [{ ...variants, active: true }] }),
      'PUT /api/v1/modules/variants': apiError(
        409,
        'module_required_by',
        'No se puede desactivar Variantes: el módulo Lotes depende de él.',
        { module: 'lots_expiry' },
      ),
    });
    renderApp('/configuracion/modulos');

    await userEvent.click(await screen.findByRole('checkbox', { name: 'Variantes' }));

    expect(await screen.findByText('No se puede desactivar Variantes: el módulo Lotes depende de él.')).toBeInTheDocument();
    expect(screen.getByRole('checkbox', { name: 'Variantes' })).toBeChecked();
  });

  it('muestra de qué módulos depende cada uno', async () => {
    fakeApi({
      'GET /api/v1/me': json(200, aUser('owner')),
      'GET /api/v1/modules': json(200, {
        items: [variants, { id: 'lots_expiry', name: 'Lotes y vencimientos', depends_on: ['variants'], active: false }],
      }),
    });
    renderApp('/configuracion/modulos');

    expect(await screen.findByRole('checkbox', { name: 'Lotes y vencimientos' })).toHaveAccessibleDescription(
      'Necesita: Variantes',
    );
  });
});
