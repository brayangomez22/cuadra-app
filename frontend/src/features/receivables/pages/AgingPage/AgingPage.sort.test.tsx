import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser } from '@/test/fakeApi';
import { renderApp } from '@/test/renderApp';
import { receivablesApi } from '../../testing/receivablesApi';

const agingKey = 'GET /api/v1/receivables/aging';

describe('AgingPage: orden y exportación', () => {
  it('ordena por lo que deben en total y empieza en la primera página', async () => {
    const api = receivablesApi();
    renderApp('/cartera?pagina=2');

    await userEvent.click(await screen.findByRole('combobox', { name: 'Ordenar por' }));
    await userEvent.click(await screen.findByRole('option', { name: 'Debe más' }));

    await waitFor(() => {
      const asked = new URL(api.calls(agingKey).at(-1)!.url);
      expect(asked.searchParams.get('sort')).toBe('total');
      expect(asked.searchParams.get('offset')).toBe('0');
    });
  });

  it('el administrador, que lee los reportes, exporta la cartera', async () => {
    receivablesApi();
    renderApp('/cartera');
    expect(await screen.findByRole('button', { name: 'Exportar a Excel' })).toBeInTheDocument();
  });

  it('el cajero ve la cartera sin exportar', async () => {
    receivablesApi({ user: aUser('cashier') });
    renderApp('/cartera');
    expect(await screen.findByRole('combobox', { name: 'Ordenar por' })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Exportar a Excel' })).not.toBeInTheDocument();
  });
});
