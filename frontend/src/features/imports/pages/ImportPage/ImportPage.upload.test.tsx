import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser, apiError, json, signedIn, fakeApi } from '@/test/fakeApi';
import { ids } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';
import { importApi } from '../../testing/importApi';
import { aFile, chooseFile, pickLocation, sentFields, validate } from '../../testing/importSteps';

function stubDownload() {
  const createObjectURL = vi.fn(() => 'blob:plantilla');
  vi.stubGlobal('URL', Object.assign(URL, { createObjectURL, revokeObjectURL: vi.fn() }));
  const downloads: string[] = [];
  vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (this: HTMLAnchorElement) {
    downloads.push(this.download);
  });
  return { createObjectURL, downloads };
}

describe('ImportPage: subir el archivo', () => {
  it('descarga la plantilla en xlsx o csv con la sesión del usuario', async () => {
    const api = importApi({ 'GET /api/v1/import-template': new Response('sku;nombre', { status: 200 }) });
    const { createObjectURL, downloads } = stubDownload();
    renderApp('/productos/importar');

    await userEvent.click(await screen.findByRole('button', { name: 'Plantilla .xlsx' }));
    await userEvent.click(screen.getByRole('button', { name: 'Plantilla .csv' }));

    await waitFor(() => expect(downloads).toEqual(['plantilla-productos.xlsx', 'plantilla-productos.csv']));
    const formats = api.calls('GET /api/v1/import-template').map((r) => new URL(r.url).searchParams.get('format'));
    expect(formats).toEqual(['xlsx', 'csv']);
    expect(createObjectURL).toHaveBeenCalledWith(expect.any(Blob));
  });

  it('sube el archivo con la sede y la política elegidas', async () => {
    const api = importApi();
    renderApp('/productos/importar');

    await pickLocation('Sede norte');
    await userEvent.click(screen.getByRole('combobox', { name: 'Si el SKU ya existe' }));
    await userEvent.click(await screen.findByRole('option', { name: 'Dejar el producto como está' }));
    await chooseFile(aFile('inventario.csv'));
    await validate();

    await screen.findByRole('table', { name: 'Primeros productos del archivo' });
    const fields = await sentFields(api.calls('POST /api/v1/imports')[0]!);
    expect(fields).toMatchObject({ location_id: ids.north, on_existing_sku: 'skip' });
    expect(Object.keys(fields)).toContain('file');
  });

  it('un archivo que no es xlsx ni csv se rechaza antes de subirlo', async () => {
    const api = importApi();
    renderApp('/productos/importar');

    await pickLocation('Bodega principal');
    await chooseFile(aFile('catalogo.pdf'));
    await validate();

    expect(await screen.findByLabelText('Archivo')).toHaveAccessibleDescription(
      expect.stringContaining('El archivo debe ser .xlsx o .csv'),
    );
    expect(api.calls('POST /api/v1/imports')).toHaveLength(0);
  });

  it('un error de la API (archivo ilegible) queda en su campo', async () => {
    importApi({
      'POST /api/v1/imports': apiError(422, { code: 'unreadable_file', message: 'No pudimos leer el archivo.' }),
    });
    renderApp('/productos/importar');

    await pickLocation('Bodega principal');
    await chooseFile();
    await validate();

    await waitFor(() =>
      expect(screen.getByLabelText('Archivo')).toHaveAccessibleDescription(
        expect.stringContaining('No pudimos leer el archivo.'),
      ),
    );
  });

  it('sin sedes activas, lleva a crear una', async () => {
    fakeApi({ ...signedIn(aUser()), 'GET /api/v1/locations': json(200, { items: [] }) });
    renderApp('/productos/importar');

    expect(await screen.findByText('Aún no hay sedes activas')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Ir a inventario' })).toHaveAttribute('href', '/inventario');
  });
});
