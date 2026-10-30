import { act, screen, waitFor } from '@testing-library/react';
import { session } from '@/shared/api/client';
import { fakeApi, signedIn, signedOut } from '@/test/fakeApi';
import { renderApp } from '@/test/renderApp';

describe('RequireAuth', () => {
  it('una ruta protegida redirige al login sin sesión', async () => {
    fakeApi(signedOut());
    const { router } = renderApp('/productos');

    await waitFor(() => expect(router.state.location.pathname).toBe('/ingresar'));
    expect(await screen.findByRole('heading', { level: 1, name: 'Ingresar a Cuadra' })).toBeInTheDocument();
  });

  it('con sesión muestra la ruta protegida', async () => {
    fakeApi(signedIn());
    const { router } = renderApp('/productos');

    expect(await screen.findByRole('heading', { level: 1, name: 'Productos' })).toBeInTheDocument();
    expect(router.state.location.pathname).toBe('/productos');
  });

  it('si la sesión termina (falla el refresh) vuelve al login', async () => {
    fakeApi(signedIn());
    const { router } = renderApp('/productos');
    await screen.findByRole('heading', { level: 1, name: 'Productos' });

    act(() => session.end());

    await waitFor(() => expect(router.state.location.pathname).toBe('/ingresar'));
  });
});
