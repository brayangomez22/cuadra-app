import { screen, within } from '@testing-library/react';
import { aUser } from '@/test/fakeApi';
import { aBoutique, aBuilder } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { customersApi } from '../../testing/customersApi';

const rowOf = async (name: string) =>
  within((await screen.findAllByRole('cell', { name: new RegExp(name) }))[0]!.closest('tr')!);

describe('CustomerListPage: lo que debe cada cliente', () => {
  it('dice lo que debe cada cliente y lo vencido, y lleva a ver quién debe más', async () => {
    customersApi({
      customers: [
        { ...aBuilder(), balance: '800000.00', overdue: '120000.00' },
        { ...aBoutique(), balance: '0.00', overdue: '0.00' },
      ],
    });
    renderApp('/clientes');

    const builder = await rowOf('Carlos Pérez');
    expect(builder.getByText('$ 800.000')).toBeInTheDocument();
    expect(builder.getByText('Vencido $ 120.000')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Quién debe más' })).toHaveAttribute('href', '/cartera?orden=total');
  });

  it('el bodeguero, sin la cartera, no ve lo que deben', async () => {
    customersApi({ user: aUser('warehouse') });
    renderApp('/clientes');

    await rowOf('Carlos Pérez');
    expect(screen.queryByRole('columnheader', { name: 'Debe' })).not.toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'Quién debe más' })).not.toBeInTheDocument();
  });
});
