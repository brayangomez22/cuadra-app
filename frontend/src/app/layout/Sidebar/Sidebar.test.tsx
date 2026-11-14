import { cleanup, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser, fakeApi, signedIn } from '@/test/fakeApi';
import { renderApp } from '@/test/renderApp';

async function mainMenu() {
  return within(await screen.findByRole('navigation', { name: 'Menú principal' }));
}

async function hideMenu() {
  await userEvent.click(await screen.findByRole('button', { name: 'Ocultar menú' }));
}

describe('Sidebar', () => {
  beforeEach(() => {
    localStorage.clear();
    fakeApi(signedIn(aUser('owner')));
  });

  it('oculta el menú lateral y lo vuelve a mostrar', async () => {
    renderApp('/');

    await hideMenu();
    const show = screen.getByRole('button', { name: 'Mostrar menú' });
    expect(show).toHaveAttribute('aria-expanded', 'false');

    await userEvent.click(show);
    expect(screen.getByRole('button', { name: 'Ocultar menú' })).toHaveAttribute('aria-expanded', 'true');
  });

  it('oculto, cada opción conserva su nombre y lo muestra al pasar el mouse', async () => {
    renderApp('/');
    await hideMenu();

    const sell = (await mainMenu()).getByRole('link', { name: 'Vender' });
    expect(sell).toHaveAttribute('title', 'Vender');
  });

  it('oculto, deja cerrar sesión', async () => {
    renderApp('/');
    await hideMenu();

    expect(screen.getByRole('button', { name: 'Cerrar sesión' })).toBeInTheDocument();
  });

  it('abierto deja elegir el tema y oculto, cambiarlo con un botón', async () => {
    renderApp('/');

    expect(await screen.findByRole('group', { name: 'Tema' })).toBeInTheDocument();
    await hideMenu();
    expect(screen.getByRole('button', { name: /^Usar tema/ })).toBeInTheDocument();
  });

  it('recuerda el menú oculto al volver a entrar', async () => {
    renderApp('/');
    await hideMenu();
    cleanup();

    fakeApi(signedIn(aUser('owner')));
    renderApp('/');
    expect(await screen.findByRole('button', { name: 'Mostrar menú' })).toBeInTheDocument();
  });

  it('abierto, las opciones no repiten su nombre como tooltip', async () => {
    renderApp('/');

    const sell = (await mainMenu()).getByRole('link', { name: 'Vender' });
    expect(sell).not.toHaveAttribute('title');
  });
});
