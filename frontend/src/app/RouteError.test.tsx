import * as Sentry from '@sentry/react';
import { render, screen } from '@testing-library/react';
import { createMemoryRouter } from 'react-router';
import { RouterProvider } from 'react-router/dom';
import { routes } from './router';

vi.mock('@sentry/react', () => ({ captureException: vi.fn() }));

function Broken(): never {
  throw new Error('Cannot read properties of undefined');
}

// renderBroken mounts a failing screen under the app's real root route, so the
// test covers the errorElement the router actually uses.
function renderBroken(path = '/roto') {
  const errorElement = routes[0]?.errorElement;
  const router = createMemoryRouter([{ errorElement, children: [{ path: '/roto', element: <Broken /> }] }], {
    initialEntries: [path],
  });
  render(<RouterProvider router={router} />);
}

describe('RouteError', () => {
  beforeEach(() => {
    // React logs the error a route threw; the test expects it.
    vi.spyOn(console, 'error').mockImplementation(() => {});
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.clearAllMocks();
  });

  it('una pantalla que falla muestra la página de error y ofrece recargar', async () => {
    renderBroken();

    expect(await screen.findByRole('heading', { name: 'Algo salió mal' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Recargar la página' })).toBeInTheDocument();
  });

  it('reporta el error a Sentry', async () => {
    renderBroken();

    await screen.findByRole('heading', { name: 'Algo salió mal' });
    expect(Sentry.captureException).toHaveBeenCalledWith(
      expect.objectContaining({ message: 'Cannot read properties of undefined' }),
    );
  });

  it('una dirección que no existe muestra que no se encontró y no la reporta', async () => {
    renderBroken('/no-existe');

    expect(await screen.findByRole('heading', { name: 'No encontramos esta página' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Ir al inicio' })).toHaveAttribute('href', '/');
    expect(Sentry.captureException).not.toHaveBeenCalled();
  });
});
