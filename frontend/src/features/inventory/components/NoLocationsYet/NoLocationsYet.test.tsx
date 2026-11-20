import { screen } from '@testing-library/react';
import { aUser, fakeApi, json, signedIn } from '@/test/fakeApi';
import { page } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';

describe('NoLocationsYet', () => {
  it.each([
    { url: '/inventario/reponer', title: 'Aún no hay sedes' },
    { url: '/inventario/conteos', title: 'Aún no hay sedes' },
    { url: '/caja', title: 'No hay sedes activas' },
  ])('en $url, el dueño va a crear la primera sede', async ({ url, title }) => {
    fakeApi({
      ...signedIn(aUser('owner')),
      'GET /api/v1/locations': json(200, { items: [] }),
      'GET /api/v1/cash-sessions/current': json(200, { required: true }),
    });
    renderApp(url);

    expect(await screen.findByRole('heading', { name: title })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Crear sede' })).toHaveAttribute('href', '/inventario');
  });

  it('quien no crea sedes pide al dueño que la cree', async () => {
    fakeApi({
      ...signedIn(aUser('admin')),
      'GET /api/v1/locations': json(200, { items: [] }),
      'GET /api/v1/inventory/reorder-alerts': json(200, page([])),
    });
    renderApp('/inventario/reponer');

    expect(await screen.findByText(/Pida al dueño que cree o active una/)).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'Crear sede' })).not.toBeInTheDocument();
  });
});
