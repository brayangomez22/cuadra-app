import { act, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { json } from '@/test/fakeApi';
import { query } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import type { Import } from '../../api/types';
import { aRowError, anImport, counts, importApi, importId } from '../../testing/importApi';

const url = `/productos/importar/${importId}`;

// replies answers GET /imports/{id} with each import in turn, and the last one from then on.
function replies(...imports: Import[]) {
  let call = 0;
  return () => json(200, imports[Math.min(call++, imports.length - 1)]!);
}

describe('ImportStatusPage', () => {
  afterEach(() => vi.useRealTimers());

  it('confirmar muestra el avance hasta el reporte final', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    importApi({
      'GET /api/v1/imports/{importId}': replies(
        anImport({ status: 'running', counts: counts({ products: 4, created: 1, updated: 1 }) }),
        anImport({ status: 'completed', counts: counts({ products: 4, created: 2, updated: 1, skipped: 1 }) }),
      ),
    });
    renderApp(url);

    const progress = await screen.findByRole('progressbar', { name: 'Avance de la importación' });
    expect(progress).toHaveAttribute('value', '2');
    expect(progress).toHaveAttribute('max', '4');
    expect(screen.getByText('2 de 4 productos')).toBeInTheDocument();

    await act(() => vi.advanceTimersByTimeAsync(1500));

    expect(await screen.findByRole('heading', { name: 'Importación terminada' })).toBeInTheDocument();
    const report = within(screen.getByRole('list', { name: 'Resultado de la importación' }));
    expect(report.getByText('2 creados')).toBeInTheDocument();
    expect(report.getByText('1 actualizado')).toBeInTheDocument();
    expect(report.getByText('1 omitido')).toBeInTheDocument();
    expect(screen.queryByRole('progressbar')).not.toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Ver productos' })).toHaveAttribute('href', '/productos');
  });

  it('una importación fallida muestra su error y todos los errores paginados', async () => {
    const api = importApi({
      'GET /api/v1/imports/{importId}': json(200, anImport({ status: 'failed', counts: counts({ errors: 25 }) })),
      'GET /api/v1/imports/{importId}/errors': (request) => {
        const offset = Number(query(request).get('offset') ?? 0);
        const row = offset === 0 ? 2 : 40;
        return json(200, {
          items: [aRowError(row, 'sku', `Fila ${String(row)}: el SKU ya lo tomó otro producto.`)],
          total: 25,
        });
      },
    });
    renderApp(url);

    expect(await screen.findByRole('alert')).toHaveTextContent('La importación no terminó');
    const errors = within(await screen.findByRole('table', { name: 'Errores del archivo' }));
    expect(errors.getByText('Fila 2: el SKU ya lo tomó otro producto.')).toBeInTheDocument();

    await userEvent.click(screen.getByRole('button', { name: 'Siguiente' }));

    expect(await screen.findByText('Fila 40: el SKU ya lo tomó otro producto.')).toBeInTheDocument();
    const offsets = api.calls(`GET /api/v1/imports/${importId}/errors`).map((r) => query(r).get('offset'));
    expect(offsets).toEqual(['0', '20']);
    expect(screen.getByRole('link', { name: 'Subir el archivo otra vez' })).toHaveAttribute(
      'href',
      '/productos/importar',
    );
  });

  it('una importación que no existe muestra el error', async () => {
    importApi();
    renderApp(url);

    expect(await screen.findByText('No pudimos cargar la importación')).toBeInTheDocument();
  });
});
