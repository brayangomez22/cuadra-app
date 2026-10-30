import { screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser, fakeApi, json } from '@/test/fakeApi';
import { ids, page, plain, query } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { aStockCount, anApprovedCount, countId, countScreenApi } from '../../testing/stockCountApi';

const listKey = 'GET /api/v1/stock-counts';
const categories = [
  { id: ids.plumbing, name: 'Plomería', created_at: '2026-10-01T15:00:00Z' },
  { id: '01920000-0000-7000-8000-000000000302', name: 'Camisetas', created_at: '2026-10-01T15:00:00Z' },
];

function countsApi(user = aUser(), counts = page([aStockCount(), anApprovedCount()])) {
  return fakeApi({
    ...countScreenApi(user),
    [listKey]: json(200, counts),
    'GET /api/v1/categories': json(200, { items: categories }),
    'POST /api/v1/stock-counts': json(201, aStockCount({ counted_lines: 0 })),
    'GET /api/v1/stock-counts/{stockCountId}/entries': json(200, page([])),
  });
}

describe('StockCountsPage', () => {
  it('muestra el conteo abierto de la sede y el historial con su exactitud', async () => {
    const api = countsApi();
    renderApp('/inventario/conteos');

    const open = await screen.findByRole('region', { name: 'Conteo abierto' });
    expect(within(open).getByText('12 de 120 productos contados')).toBeInTheDocument();
    expect(within(open).getByRole('link', { name: 'Contar' })).toHaveAttribute(
      'href',
      `/inventario/conteos/${countId}`,
    );
    expect(within(open).getByRole('link', { name: 'Ver diferencias' })).toHaveAttribute(
      'href',
      `/inventario/conteos/${countId}/diferencias`,
    );

    const history = within(screen.getByRole('table', { name: 'Conteos de Bodega principal' }));
    const approved = history.getByRole('row', { name: /Aprobado/ });
    expect(plain(approved.textContent)).toContain('95 %');
    expect(plain(approved.textContent)).toContain('-$ 48.500');
    expect(query(api.calls(listKey)[0]!).get('location_id')).toBe(ids.main);
  });

  it('nombra las categorías de un conteo parcial en el historial', async () => {
    countsApi(aUser(), page([anApprovedCount({ category_ids: [ids.plumbing] })]));
    renderApp('/inventario/conteos');

    const history = within(await screen.findByRole('table', { name: 'Conteos de Bodega principal' }));
    expect(await history.findByRole('cell', { name: 'Plomería' })).toBeInTheDocument();
  });

  it('inicia un conteo parcial por categorías en modo revisión y abre la pantalla de conteo', async () => {
    const api = countsApi(aUser(), page([anApprovedCount()]));
    renderApp('/inventario/conteos');

    await userEvent.click(await screen.findByRole('button', { name: 'Iniciar conteo' }));
    const dialog = within(await screen.findByRole('dialog', { name: 'Iniciar conteo en Bodega principal' }));
    await userEvent.click(dialog.getByRole('combobox', { name: 'Qué contar' }));
    await userEvent.click(await screen.findByRole('option', { name: 'Solo algunas categorías' }));
    await userEvent.click(await dialog.findByRole('checkbox', { name: 'Plomería' }));
    await userEvent.click(dialog.getByRole('combobox', { name: 'Si dos personas cuentan el mismo producto' }));
    await userEvent.click(await screen.findByRole('option', { name: /Deben coincidir/ }));
    await userEvent.click(dialog.getByRole('button', { name: 'Iniciar conteo' }));

    expect(await screen.findByRole('heading', { name: 'Conteo de Bodega principal' })).toBeInTheDocument();
    expect(await api.calls('POST /api/v1/stock-counts')[0]!.json()).toEqual({
      location_id: ids.main,
      category_ids: [ids.plumbing],
      mode: 'review',
    });
  });

  it('quien cuenta no inicia conteos ni ve las diferencias', async () => {
    countsApi(aUser('cashier'));
    renderApp('/inventario/conteos');

    const open = await screen.findByRole('region', { name: 'Conteo abierto' });
    expect(within(open).getByRole('link', { name: 'Contar' })).toBeInTheDocument();
    expect(within(open).queryByRole('link', { name: 'Ver diferencias' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Iniciar conteo' })).not.toBeInTheDocument();
  });

  it('sin conteos lo dice e invita a iniciar uno', async () => {
    countsApi(aUser(), page([]));
    renderApp('/inventario/conteos');

    expect(await screen.findByRole('heading', { name: 'Aún no hay conteos en Bodega principal' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Iniciar conteo' })).toBeInTheDocument();
  });
});
