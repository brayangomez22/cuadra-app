import { screen, waitFor, within } from '@testing-library/react';
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

  it('pide contar todo el efectivo del cajón, base incluida, y nada más que efectivo', async () => {
    closeApi();
    renderApp('/caja/cerrar');

    const instructions = await screen.findByText(/Cuente todo el efectivo del cajón/);
    expect(plain(instructions.closest('p')?.textContent)).toBe(
      'Cuente todo el efectivo del cajón, incluida la base de $ 100.000 con la que abrió. ' +
        'No reste nada: Cuadra hace las cuentas al cerrar. ' +
        'Los váuchers de tarjeta y los comprobantes de transferencias no se cuentan.',
    );
  });

  it('agrupa el conteo en billetes y monedas, como se cuenta el cajón', async () => {
    closeApi();
    renderApp('/caja/cerrar');

    const bills = within(await screen.findByRole('group', { name: 'Billetes' }));
    expect(bills.getByLabelText('Billetes de $ 100.000')).toBeInTheDocument();
    expect(bills.getByLabelText('Billetes de $ 2.000')).toBeInTheDocument();
    const coins = within(screen.getByRole('group', { name: 'Monedas' }));
    expect(coins.getByLabelText('Billetes y monedas de $ 1.000')).toBeInTheDocument();
    expect(coins.getByLabelText('Monedas de $ 50')).toBeInTheDocument();
  });
});
