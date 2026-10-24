import { screen, waitFor } from '@testing-library/react';
import { plain } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { CLOSE_KEY, closeApi } from '../../testing/cashApi';
import { clearPosStorage, user } from '../../testing/posSteps';

beforeEach(clearPosStorage);

describe('CloseCashPage', () => {
  it('suma lo contado por denominación mientras se escribe, sin mostrar el esperado', async () => {
    closeApi();
    renderApp('/caja/cerrar');

    await user().type(await screen.findByLabelText('Billetes de $ 100.000'), '1');
    await user().type(screen.getByLabelText('Billetes de $ 20.000'), '2');
    await user().type(screen.getByLabelText('Monedas de $ 500'), '3');

    const total = screen.getByRole('status', { name: 'Total contado' });
    await waitFor(() => expect(plain(total.textContent)).toContain('$ 141.500'));
    expect(screen.queryByText(/esperado/i)).not.toBeInTheDocument();
  });

  it('rechaza una cantidad que no es un número entero', async () => {
    const api = closeApi();
    renderApp('/caja/cerrar');

    await user().type(await screen.findByLabelText('Billetes de $ 50.000'), '1,5');
    await user().click(screen.getByRole('button', { name: 'Cerrar caja' }));

    expect(await screen.findByText('Escriba cuántos hay, sin decimales')).toBeInTheDocument();
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    expect(api.calls(CLOSE_KEY)).toHaveLength(0);
  });
});
