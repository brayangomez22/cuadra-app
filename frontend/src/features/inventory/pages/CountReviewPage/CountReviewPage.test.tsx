import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser, apiError, fakeApi, json, type Reply } from '@/test/fakeApi';
import { ids, plain, query, variantId } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import type { ResponseBody } from '@/test/routeKeys';
import { aCountLine, aStockCount, countId, countScreenApi } from '../../testing/stockCountApi';

const linesKey = 'GET /api/v1/stock-counts/{stockCountId}/lines';
const approveKey = 'POST /api/v1/stock-counts/{stockCountId}/approve';
const url = `/inventario/conteos/${countId}/diferencias`;
const linesCalls = `GET /api/v1/stock-counts/${countId}/lines`;
const recountCalls = `POST /api/v1/stock-counts/${countId}/recount`;

const lines = {
  items: [
    aCountLine({
      variant_id: variantId('M', 'Azul'),
      product_id: ids.garment,
      status: 'review',
      expected_quantity: '10',
      counted_quantity: undefined,
      difference: undefined,
      difference_value: undefined,
    }),
    aCountLine(),
  ],
  total: 2,
  summary: {
    lines: 120,
    counted_lines: 100,
    exact_lines: 95,
    accuracy: '0.9500',
    net_difference_value: '-30000.00',
    absolute_difference_value: '30000.00',
  },
};

function reviewApi(approve: Reply<ResponseBody<typeof approveKey>> = json(200, aStockCount({ status: 'approved' }))) {
  return fakeApi({
    ...countScreenApi(aUser('admin')),
    [linesKey]: json(200, lines),
    [approveKey]: approve,
    'POST /api/v1/stock-counts/{stockCountId}/recount': new Response(null, { status: 204 }),
    'GET /api/v1/stock-counts': json(200, { items: [], total: 0 }),
  });
}

describe('CountReviewPage', () => {
  it('muestra lo esperado, lo contado y la diferencia con su valor, y el resumen', async () => {
    const api = reviewApi();
    renderApp(url);

    const table = within(await screen.findByRole('table', { name: 'Diferencias del conteo' }));
    const pipe = await table.findByRole('row', { name: /Tubo PVC/ });
    expect(plain(pipe.textContent)).toMatch(/Con diferencia.*50.*47,5.*-2,5.*-\$ 30\.000/);
    expect(plain(table.getByRole('row', { name: /Camiseta básica \(Azul · M\)/ }).textContent)).toContain(
      'En revisión',
    );
    expect(plain(screen.getByRole('region', { name: 'Resumen del conteo' }).textContent)).toMatch(/95 %.*100 de 120/);
    expect(query(api.calls(linesCalls)[0]!).get('status')).toBeNull();
  });

  it('filtra por estado desde la URL', async () => {
    const api = reviewApi();
    renderApp(`${url}?estado=revision`);

    await screen.findByRole('table', { name: 'Diferencias del conteo' });
    expect(query(api.calls(linesCalls)[0]!).get('status')).toBe('review');
  });

  it('pide recontar una variante en revisión', async () => {
    const api = reviewApi();
    renderApp(url);

    await userEvent.click(await screen.findByRole('button', { name: /Recontar Camiseta básica/ }));
    await waitFor(() => expect(api.calls(recountCalls)).toHaveLength(1));
    expect(await api.calls(recountCalls)[0]!.json()).toEqual({
      variant_ids: [variantId('M', 'Azul')],
    });
  });

  it('aprobar con variantes en revisión muestra el error y no cierra', async () => {
    reviewApi(
      apiError(409, {
        code: 'stock_count_needs_review',
        message: 'Hay productos en revisión: pida recontarlos antes de aprobar.',
      }),
    );
    renderApp(url);

    await userEvent.click(await screen.findByRole('button', { name: 'Aprobar conteo' }));
    await userEvent.click(within(await screen.findByRole('dialog')).getByRole('button', { name: 'Aprobar y ajustar' }));

    expect(
      await screen.findByText('Hay productos en revisión: pida recontarlos antes de aprobar.'),
    ).toBeInTheDocument();
    expect(screen.getByRole('dialog', { name: 'Aprobar el conteo' })).toBeInTheDocument();
  });

  it('aprueba el conteo y vuelve a la lista de conteos', async () => {
    const api = reviewApi();
    renderApp(url);

    await userEvent.click(await screen.findByRole('button', { name: 'Aprobar conteo' }));
    await userEvent.click(within(await screen.findByRole('dialog')).getByRole('button', { name: 'Aprobar y ajustar' }));

    expect(await screen.findByRole('heading', { name: 'Toma física' })).toBeInTheDocument();
    expect(api.calls(`POST /api/v1/stock-counts/${countId}/approve`)).toHaveLength(1);
  });
  describe('sin stock para ajustar (P90)', () => {
    const noStock = () =>
      reviewApi(
        apiError(409, {
          code: 'insufficient_stock',
          message: 'Un ajuste dejaría sin stock un producto.',
          details: { variant_id: ids.pipe },
        }),
      );

    async function approve() {
      await userEvent.click(await screen.findByRole('button', { name: 'Aprobar conteo' }));
      await userEvent.click(
        within(await screen.findByRole('dialog')).getByRole('button', { name: 'Aprobar y ajustar' }),
      );
      return within(await screen.findByRole('dialog', { name: 'Aprobar el conteo' }));
    }

    it('al aprobar nombra el producto que se quedaría sin stock', async () => {
      noStock();
      renderApp(url);

      const dialog = await approve();

      expect(await dialog.findByText(/No hay stock para ajustar Tubo PVC presión 1\/2 pulgada/)).toBeInTheDocument();
      expect(dialog.getByRole('button', { name: 'Recontar Tubo PVC presión 1/2 pulgada' })).toBeInTheDocument();
    });

    it('recontar desde el error lo deja marcado en revisión', async () => {
      const api = noStock();
      renderApp(url);

      const dialog = await approve();
      await userEvent.click(await dialog.findByRole('button', { name: /^Recontar Tubo PVC/ }));

      await waitFor(() => expect(api.calls(recountCalls)).toHaveLength(1));
      expect(await api.calls(recountCalls)[0]!.json()).toEqual({ variant_ids: [ids.pipe] });
      await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
      await waitFor(() => expect(query(api.calls(linesCalls).at(-1)!).get('status')).toBe('review'));
      const row = await screen.findByRole('row', { name: /Tubo PVC/ });
      expect(row).toHaveAttribute('aria-current', 'true');
    });
  });
});
