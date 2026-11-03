import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { todayInBogota } from '@/shared/format';
import { json } from '@/test/fakeApi';
import { aLocation, plain, query } from '@/test/fixtures';
import { preloadPages } from '@/test/preloadPages';
import { renderApp } from '@/test/renderApp';
import { aSalePage, aSaleSummary, aVoidedSale, saleId, salesApi } from '../../testing/salesApi';

const cells = (row: HTMLElement) =>
  within(row)
    .getAllByRole('cell')
    .map((cell) => plain(cell.textContent));

beforeAll(preloadPages('/ventas'));

const otherLocation = '01920000-0000-7000-8000-000000000a02';

describe('SaleListPage', () => {
  it('muestra las ventas de hoy, marca solo las anuladas y enlaza a su detalle', async () => {
    const voided = aVoidedSale({ id: '01920000-0000-7000-8000-000000000502', number: 14 });
    const api = salesApi({
      'GET /api/v1/sales': json(
        200,
        aSalePage([aSaleSummary(), aSaleSummary(voided)], { voided: 1, sold: '11900.00' }),
      ),
    });
    renderApp('/ventas');

    const table = within(await screen.findByRole('table', { name: 'Ventas' }));
    expect(query(api.calls('GET /api/v1/sales')[0]!).get('date')).toBe(todayInBogota());
    expect(table.getByRole('link', { name: '15' })).toHaveAttribute('href', `/ventas/${saleId}`);
    expect(cells(table.getByRole('row', { name: /^15/ })).join()).not.toContain('Confirmada');
    expect(cells(table.getByRole('row', { name: /^14/ })).at(-1)).toContain('Anulada');
    expect(cells(table.getByRole('row', { name: /^15/ })).join()).toContain('10:00 a. m.');
    expect(cells(table.getByRole('row', { name: /^15/ })).join()).toContain('$ 11.900');
  });

  it('resume el día con todas sus páginas: cuántas ventas, lo vendido y las anuladas', async () => {
    salesApi({
      'GET /api/v1/sales': json(200, aSalePage([aSaleSummary()], { total: 91, voided: 2, sold: '24518300.00' })),
    });
    renderApp('/ventas?fecha=2026-10-11');

    const summary = within(await screen.findByRole('region', { name: 'Resumen del día' }));
    expect(plain(summary.getByText('Ventas').nextElementSibling?.textContent)).toBe('91');
    expect(plain(summary.getByText('Vendido').nextElementSibling?.textContent)).toBe('$ 24.518.300');
    expect(plain(summary.getByText('Anuladas').nextElementSibling?.textContent)).toBe('2');
  });

  it('lee el día y la sede de la URL y los manda a la API', async () => {
    const api = salesApi({
      'GET /api/v1/locations': json(200, {
        items: [aLocation(), aLocation({ id: otherLocation, name: 'Sede Norte' })],
      }),
    });
    renderApp(`/ventas?fecha=2026-10-11&sede=${otherLocation}`);

    await screen.findByRole('table', { name: 'Ventas' });
    const asked = query(api.calls('GET /api/v1/sales')[0]!);
    expect(asked.get('date')).toBe('2026-10-11');
    expect(asked.get('location_id')).toBe(otherLocation);
  });

  it('cambiar de día vuelve a la primera página', async () => {
    const api = salesApi({ 'GET /api/v1/sales': json(200, aSalePage([aSaleSummary()], { total: 45 })) });
    renderApp('/ventas?fecha=2026-10-11&pagina=2');

    await screen.findByRole('table', { name: 'Ventas' });
    expect(query(api.calls('GET /api/v1/sales')[0]!).get('offset')).toBe('20');
    fireEvent.change(screen.getByLabelText('Día'), { target: { value: '2026-10-10' } });

    await waitFor(() => {
      const last = query(api.calls('GET /api/v1/sales').at(-1)!);
      expect(last.get('date')).toBe('2026-10-10');
      expect(last.get('offset')).toBe('0');
    });
  });

  it('un día sin ventas lo dice con la fecha y lleva al día anterior', async () => {
    salesApi({ 'GET /api/v1/sales': json(200, aSalePage([])) });
    renderApp('/ventas?fecha=2026-10-11');

    expect(await screen.findByText('No hay ventas el 11/10/2026')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Ver el 10/10/2026' })).toHaveAttribute('href', '/ventas?fecha=2026-10-10');
  });

  it('muestra la sede solo si hay más de una para elegir', async () => {
    salesApi();
    renderApp('/ventas');

    await screen.findByRole('table', { name: 'Ventas' });
    expect(screen.queryByRole('combobox', { name: 'Sede' })).not.toBeInTheDocument();
  });
});
