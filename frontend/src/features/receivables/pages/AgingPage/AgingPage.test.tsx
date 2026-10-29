import { screen, within } from '@testing-library/react';
import { json } from '@/test/fakeApi';
import { buckets, customerIds, plain } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { receivablesApi } from '../../testing/receivablesApi';

const cells = (row: HTMLElement) =>
  within(row)
    .getAllByRole('cell')
    .map((cell) => plain(cell.textContent));

describe('AgingPage', () => {
  it('muestra lo que debe cada cliente por edad y los totales', async () => {
    receivablesApi();
    renderApp('/cartera');

    const table = within(await screen.findByRole('table', { name: /Cartera por edades/ }));
    expect(cells(table.getByRole('row', { name: /Carlos Pérez/ }))).toEqual([
      'Carlos Pérez',
      '$ 200.000',
      '$ 150.000',
      '$ 0',
      '$ 0',
      '$ 0',
      '$ 350.000',
    ]);
    expect(cells(table.getByRole('row', { name: /Confecciones Sol SAS/ })).at(-2)).toBe('$ 80.000');
    expect(cells(table.getByRole('row', { name: /^Total/ }))).toEqual([
      'Total',
      '$ 200.000',
      '$ 150.000',
      '$ 0',
      '$ 0',
      '$ 80.000',
      '$ 430.000',
    ]);
    expect(screen.getByText(/28\/10\/2026/)).toBeInTheDocument();
    expect(plain(screen.getByText('Vencida', { selector: 'dt' }).nextElementSibling?.textContent)).toBe('$ 230.000');
  });

  it('cada cliente lleva a su cuenta', async () => {
    receivablesApi();
    renderApp('/cartera');

    expect(await screen.findByRole('link', { name: 'Confecciones Sol SAS' })).toHaveAttribute(
      'href',
      `/clientes/${customerIds.boutique}/cuenta`,
    );
  });

  it('sin cartera lo dice', async () => {
    receivablesApi({
      routes: {
        'GET /api/v1/receivables/aging': json(200, { as_of: '2026-10-28', items: [], total: 0, totals: buckets({}) }),
      },
    });
    renderApp('/cartera');

    expect(await screen.findByText('Nadie debe a la empresa')).toBeInTheDocument();
  });
});
