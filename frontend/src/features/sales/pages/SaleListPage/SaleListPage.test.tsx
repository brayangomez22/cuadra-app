import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { todayInBogota } from '@/shared/format';
import { json } from '@/test/fakeApi';
import { aLocation, page, plain, query } from '@/test/fixtures';
import { preloadPages } from '@/test/preloadPages';
import { renderApp } from '@/test/renderApp';
import { aSaleSummary, aVoidedSale, saleId, salesApi } from '../../testing/salesApi';

const cells = (row: HTMLElement) =>
  within(row)
    .getAllByRole('cell')
    .map((cell) => plain(cell.textContent));

beforeAll(preloadPages('/ventas'));

const otherLocation = '01920000-0000-7000-8000-000000000a02';

describe('SaleListPage', () => {
  it('muestra las ventas de hoy, con las anuladas marcadas, y enlaza a su detalle', async () => {
    const voided = aVoidedSale({ id: '01920000-0000-7000-8000-000000000502', number: 14 });
    const api = salesApi({ 'GET /api/v1/sales': json(200, page([aSaleSummary(), aSaleSummary(voided)])) });
    renderApp('/ventas');

    const table = within(await screen.findByRole('table', { name: 'Ventas' }));
    expect(query(api.calls('GET /api/v1/sales')[0]!).get('date')).toBe(todayInBogota());
    expect(table.getByRole('link', { name: '15' })).toHaveAttribute('href', `/ventas/${saleId}`);
    expect(cells(table.getByRole('row', { name: /^15/ })).slice(-1)).toEqual(['Confirmada']);
    expect(cells(table.getByRole('row', { name: /^14/ })).slice(-1)).toEqual(['Anulada']);
    expect(cells(table.getByRole('row', { name: /^15/ })).join()).toContain('10:00 a. m.');
    expect(cells(table.getByRole('row', { name: /^15/ })).join()).toContain('$ 11.900');
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
    const api = salesApi({ 'GET /api/v1/sales': json(200, { items: [aSaleSummary()], total: 45 }) });
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

  it('un día sin ventas lo dice con la fecha', async () => {
    salesApi({ 'GET /api/v1/sales': json(200, page([])) });
    renderApp('/ventas?fecha=2026-10-11');

    expect(await screen.findByText('No hay ventas el 11/10/2026')).toBeInTheDocument();
  });

  it('muestra la sede solo si hay más de una para elegir', async () => {
    salesApi();
    renderApp('/ventas');

    await screen.findByRole('table', { name: 'Ventas' });
    expect(screen.queryByRole('combobox', { name: 'Sede' })).not.toBeInTheDocument();
  });
});
