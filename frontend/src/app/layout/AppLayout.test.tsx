import { cleanup, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser, fakeApi, signedIn } from '@/test/fakeApi';
import { renderApp } from '@/test/renderApp';

async function mainMenu() {
  return within(await screen.findByRole('navigation', { name: 'Menú principal' }));
}

describe('AppLayout', () => {
  it('muestra todas las opciones del menú al dueño', async () => {
    fakeApi(signedIn(aUser('owner')));
    renderApp('/');

    const menu = await mainMenu();

    for (const name of ['Inicio', 'Vender', 'Productos', 'Inventario']) {
      expect(menu.getByRole('link', { name })).toBeInTheDocument();
    }
  });

  it('muestra Configuración a quien configura el catálogo o la empresa, no al cajero', async () => {
    for (const role of ['owner', 'admin'] as const) {
      fakeApi(signedIn(aUser(role)));
      renderApp('/');
      expect((await mainMenu()).getByRole('link', { name: 'Configuración' })).toBeInTheDocument();
      cleanup();
    }

    fakeApi(signedIn(aUser('cashier')));
    renderApp('/');
    expect((await mainMenu()).queryByRole('link', { name: 'Configuración' })).not.toBeInTheDocument();
  });

  it('muestra Compras al dueño y al administrador, no al cajero ni al bodeguero', async () => {
    for (const role of ['owner', 'admin'] as const) {
      fakeApi(signedIn(aUser(role)));
      renderApp('/');
      expect((await mainMenu()).getByRole('link', { name: 'Compras' })).toBeInTheDocument();
      cleanup();
    }
    for (const role of ['cashier', 'warehouse'] as const) {
      fakeApi(signedIn(aUser(role)));
      renderApp('/');
      expect((await mainMenu()).queryByRole('link', { name: 'Compras' })).not.toBeInTheDocument();
      cleanup();
    }
  });

  it('muestra solo las opciones del menú permitidas para el rol', async () => {
    fakeApi(signedIn(aUser('warehouse')));
    renderApp('/');

    const menu = await mainMenu();

    expect(menu.getByRole('link', { name: 'Inventario' })).toBeInTheDocument();
    expect(menu.queryByRole('link', { name: 'Vender' })).not.toBeInTheDocument();
  });

  it('marca la opción de la página actual', async () => {
    fakeApi(signedIn());
    renderApp('/inventario');

    const menu = await mainMenu();

    expect(menu.getByRole('link', { name: 'Inventario' })).toHaveAttribute('aria-current', 'page');
    expect(menu.getByRole('link', { name: 'Inicio' })).not.toHaveAttribute('aria-current');
  });

  it('en una página de una opción más específica marca solo esa', async () => {
    fakeApi(signedIn());
    // Toma física vive bajo /inventario: solo ella queda marcada.
    renderApp('/inventario/conteos');

    const menu = await mainMenu();

    expect(menu.getByRole('link', { name: 'Toma física' })).toHaveAttribute('aria-current', 'page');
    expect(menu.getByRole('link', { name: 'Inventario' })).not.toHaveAttribute('aria-current');
  });

  it('muestra la empresa, el usuario y su rol', async () => {
    fakeApi(signedIn(aUser('cashier')));
    renderApp('/');

    expect(await screen.findAllByText('Ferretería El Tornillo')).not.toHaveLength(0);
    expect(screen.getAllByText('Ana Gómez')).not.toHaveLength(0);
    expect(screen.getAllByText('Cajero')).not.toHaveLength(0);
  });

  it('el botón de menú abre el panel en móvil y Escape lo cierra devolviendo el foco', async () => {
    fakeApi(signedIn());
    renderApp('/');
    const toggle = await screen.findByRole('button', { name: 'Abrir menú' });
    expect(toggle).toHaveAttribute('aria-expanded', 'false');

    await userEvent.click(toggle);

    const panel = screen.getByRole('dialog', { name: 'Menú' });
    expect(within(panel).getByRole('link', { name: 'Productos' })).toBeInTheDocument();
    expect(toggle).toHaveAttribute('aria-expanded', 'true');

    await userEvent.keyboard('{Escape}');

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    expect(toggle).toHaveFocus();
  });

  it('el panel móvil se cierra al elegir una opción', async () => {
    fakeApi(signedIn());
    const { router } = renderApp('/');
    await userEvent.click(await screen.findByRole('button', { name: 'Abrir menú' }));

    await userEvent.click(within(screen.getByRole('dialog')).getByRole('link', { name: 'Productos' }));

    // The screen loads lazily: the navigation ends once its module is in.
    await waitFor(() => expect(router.state.location.pathname).toBe('/productos'));
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
  });

  it('cerrar sesión llama a logout y lleva al login', async () => {
    const api = fakeApi({ ...signedIn(), 'POST /api/v1/auth/logout': new Response(null, { status: 204 }) });
    const { router } = renderApp('/');

    await userEvent.click((await screen.findAllByRole('button', { name: 'Cerrar sesión' }))[0]!);

    await waitFor(() => expect(router.state.location.pathname).toBe('/ingresar'));
    expect(api.calls('POST /api/v1/auth/logout')).toHaveLength(1);
  });
});
