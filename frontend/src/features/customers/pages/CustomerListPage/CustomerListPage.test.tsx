import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser } from '@/test/fakeApi';
import { query, segmentIds } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { customersApi } from '../../testing/customersApi';

const rowOf = async (name: string) => within(await screen.findByRole('row', { name: new RegExp(name) }));

describe('CustomerListPage', () => {
  it('lista los clientes con su documento y segmento', async () => {
    customersApi();
    renderApp('/clientes');

    const builder = await rowOf('Carlos Pérez');
    expect(builder.getByText('CC 1020304050')).toBeInTheDocument();
    expect(builder.getByText('Maestro de obra')).toBeInTheDocument();
    const boutique = await rowOf('Confecciones Sol SAS');
    expect(boutique.getByText('Boutique Sol')).toBeInTheDocument();
    expect(boutique.getByText('NIT 901234567-7')).toBeInTheDocument();
    expect(boutique.getByText('Mayorista')).toBeInTheDocument();
    expect(boutique.getByRole('link', { name: 'Editar Confecciones Sol SAS' })).toHaveAttribute(
      'href',
      '/clientes/01920000-0000-7000-8000-000000000502/editar',
    );
  });

  it('busca y filtra por segmento y estado desde la URL', async () => {
    const api = customersApi();
    renderApp(`/clientes?q=sol&segmento=${segmentIds.wholesale}&estado=todos`);

    await rowOf('Confecciones Sol SAS');
    expect(screen.queryByRole('row', { name: /Carlos Pérez/ })).not.toBeInTheDocument();
    const sent = query(api.calls('GET /api/v1/customers').at(-1)!);
    expect(sent.get('q')).toBe('sol');
    expect(sent.get('segment_id')).toBe(segmentIds.wholesale);
    expect(sent.get('status')).toBe('all');
    expect(screen.getByRole('searchbox', { name: 'Buscar' })).toHaveValue('sol');
  });

  it('al escribir busca en la API', async () => {
    const api = customersApi();
    renderApp('/clientes');
    await rowOf('Carlos Pérez');

    await userEvent.type(screen.getByRole('searchbox', { name: 'Buscar' }), '1020');

    await waitFor(() => expect(query(api.calls('GET /api/v1/customers').at(-1)!).get('q')).toBe('1020'));
    expect(await rowOf('Carlos Pérez')).toBeDefined();
    await waitFor(() => expect(screen.queryByRole('row', { name: /Confecciones Sol/ })).not.toBeInTheDocument());
  });

  it('sin clientes ofrece crear el primero', async () => {
    customersApi({ customers: [] });
    renderApp('/clientes');

    expect(await screen.findByText('Todavía no hay clientes')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Crear el primer cliente' })).toHaveAttribute('href', '/clientes/nuevo');
  });

  it('el bodeguero ve la lista sin crear ni editar', async () => {
    customersApi({ user: aUser('warehouse') });
    renderApp('/clientes');

    await rowOf('Carlos Pérez');
    expect(screen.queryByRole('link', { name: 'Nuevo cliente' })).not.toBeInTheDocument();
    expect(screen.queryByRole('link', { name: /^Editar/ })).not.toBeInTheDocument();
  });
});
