import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { fakeCamera } from '@/test/camera';
import { aUser, apiError, fakeApi, json } from '@/test/fakeApi';
import { aGarment, ids, page, plain, variantBarcode, variantId } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { aCountEntry, aStockCount, countId, countScreenApi } from '../../testing/stockCountApi';

const entriesKey = 'GET /api/v1/stock-counts/{stockCountId}/entries';
const recordKey = 'POST /api/v1/stock-counts/{stockCountId}/entries';
const recorded = `POST /api/v1/stock-counts/${countId}/entries`;
const url = `/inventario/conteos/${countId}`;

function countingApi(count = aStockCount(), entries = page([aCountEntry()])) {
  return fakeApi({
    ...countScreenApi(aUser('warehouse'), count),
    [entriesKey]: json(200, entries),
    [recordKey]: json(201, aCountEntry({ id: '01920000-0000-7000-8000-000000000a12', quantity: '8' })),
    'DELETE /api/v1/stock-counts/{stockCountId}/entries/{countEntryId}': new Response(null, { status: 204 }),
  });
}

async function pickPipe() {
  await userEvent.type(await screen.findByRole('searchbox', { name: 'Producto' }), 'Tubo');
  await userEvent.click(await screen.findByRole('button', { name: /TUB-PVC-12/ }));
}

describe('CountingPage', () => {
  it('registra lo contado sin mostrar lo esperado y lo lista', async () => {
    const api = countingApi();
    renderApp(url);

    expect(await screen.findByRole('heading', { name: 'Conteo de Bodega principal' })).toBeInTheDocument();
    expect(screen.getByText('12 de 120 productos contados')).toBeInTheDocument();
    await pickPipe();
    await userEvent.type(screen.getByRole('textbox', { name: 'Cantidad contada' }), '8');
    await userEvent.click(screen.getByRole('button', { name: 'Registrar' }));

    await waitFor(() => expect(api.calls(recorded)).toHaveLength(1));
    expect(await api.calls(recorded)[0]!.json()).toEqual({ variant_id: ids.pipe, quantity: '8' });
    const mine = within(screen.getByRole('list', { name: 'Lo que usted contó' }));
    expect(plain((await mine.findAllByRole('listitem'))[0]!.textContent)).toContain('Tubo PVC');
    expect(document.body.textContent).not.toMatch(/esperad/i);
  });

  it('escanea con la cámara el código de una talla y la deja lista para contar', async () => {
    const camera = fakeCamera();
    const api = countingApi();
    renderApp(url);

    await userEvent.click(await screen.findByRole('button', { name: 'Escanear con la cámara' }));
    camera.show(variantBarcode('M', 'Azul'));

    expect(await screen.findByText(/Camiseta básica \(Azul · M\)/)).toBeInTheDocument();
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    await userEvent.type(screen.getByRole('textbox', { name: 'Cantidad contada' }), '3');
    await userEvent.click(screen.getByRole('button', { name: 'Registrar' }));
    await waitFor(() => expect(api.calls(recorded)).toHaveLength(1));
    expect(await api.calls(recorded)[0]!.json()).toEqual({ variant_id: variantId('M', 'Azul'), quantity: '3' });
    expect(aGarment().id).toBe(ids.garment);
  });

  it('borra lo que contó por error', async () => {
    const api = countingApi();
    renderApp(url);

    await userEvent.click(await screen.findByRole('button', { name: /Borrar 47,5 de Tubo PVC/ }));
    await waitFor(() =>
      expect(api.calls(`DELETE /api/v1/stock-counts/${countId}/entries/${aCountEntry().id}`)).toHaveLength(1),
    );
  });

  it('un producto fuera del conteo parcial lo dice sobre el formulario', async () => {
    fakeApi({
      ...countScreenApi(aUser('cashier'), aStockCount({ category_ids: [ids.plumbing] })),
      [entriesKey]: json(200, page([])),
      [recordKey]: apiError(422, { code: 'variant_not_in_count', message: 'Ese producto no está en este conteo.' }),
    });
    renderApp(url);

    await pickPipe();
    await userEvent.type(screen.getByRole('textbox', { name: 'Cantidad contada' }), '1');
    await userEvent.click(screen.getByRole('button', { name: 'Registrar' }));
    expect(await screen.findByText('Ese producto no está en este conteo.')).toBeInTheDocument();
  });

  it('un conteo cerrado ya no deja contar', async () => {
    countingApi(aStockCount({ status: 'approved', closed_at: '2026-10-28T18:00:00Z', closed_by: aUser().id }));
    renderApp(url);

    expect(await screen.findByText('Este conteo ya se aprobó: no se puede contar más.')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Registrar' })).not.toBeInTheDocument();
  });
});
