import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { createMemoryRouter, Link } from 'react-router';
import { RouterProvider } from 'react-router/dom';
import { DiscardChangesGuard } from './DiscardChangesGuard';

function renderGuard(dirty: boolean) {
  const router = createMemoryRouter(
    [
      {
        path: '/',
        element: (
          <>
            <Link to="/otra">Salir</Link>
            <DiscardChangesGuard
              shouldBlock={() => dirty}
              title="¿Descartar el pedido?"
              description="Se pierde lo que lleva escrito."
            />
          </>
        ),
      },
      { path: '/otra', element: <p>Otra pantalla</p> },
    ],
    { initialEntries: ['/'] },
  );
  render(<RouterProvider router={router} />);
  return router;
}

describe('DiscardChangesGuard', () => {
  it('con cambios pregunta antes de salir y «Seguir editando» se queda', async () => {
    const router = renderGuard(true);

    await userEvent.click(screen.getByRole('link', { name: 'Salir' }));
    await userEvent.click(await screen.findByRole('button', { name: 'Seguir editando' }));

    expect(router.state.location.pathname).toBe('/');
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('«Descartar» sale', async () => {
    const router = renderGuard(true);

    await userEvent.click(screen.getByRole('link', { name: 'Salir' }));
    await userEvent.click(await screen.findByRole('button', { name: 'Descartar' }));

    expect(await screen.findByText('Otra pantalla')).toBeInTheDocument();
    expect(router.state.location.pathname).toBe('/otra');
  });

  it('sin cambios sale sin preguntar, y con cambios el navegador avisa al recargar', async () => {
    renderGuard(false);
    const reload = new Event('beforeunload', { cancelable: true });
    window.dispatchEvent(reload);
    expect(reload.defaultPrevented).toBe(false);

    await userEvent.click(screen.getByRole('link', { name: 'Salir' }));
    expect(await screen.findByText('Otra pantalla')).toBeInTheDocument();
  });
});
