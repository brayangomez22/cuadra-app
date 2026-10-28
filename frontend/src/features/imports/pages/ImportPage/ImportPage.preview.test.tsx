import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { json } from '@/test/fakeApi';
import { renderApp } from '@/test/renderApp';
import { aRejectedPreview, importApi, importId } from '../../testing/importApi';
import { chooseFile, confirm, pickLocation, validate } from '../../testing/importSteps';

async function upload() {
  const { router } = renderApp('/productos/importar');
  await pickLocation('Bodega principal');
  await chooseFile();
  await validate();
  return router;
}

describe('ImportPage: previsualizar y confirmar', () => {
  it('un archivo válido muestra los primeros productos con sus variantes y deja confirmar', async () => {
    importApi();
    await upload();

    const table = within(await screen.findByRole('table', { name: 'Primeros productos del archivo' }));
    expect(table.getByText('Martillo de uña 16 oz')).toBeInTheDocument();
    expect(table.getByText('CAMB-M')).toBeInTheDocument();
    expect(table.getByText('talla: L')).toBeInTheDocument();
    expect(screen.getByText(/2 productos en 3 filas/)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Importar' })).toBeEnabled();
  });

  it('un archivo con errores muestra cada error con su fila y no deja confirmar', async () => {
    importApi({ 'POST /api/v1/imports': json(201, aRejectedPreview()) });
    await upload();

    const errors = within(await screen.findByRole('table', { name: 'Errores del archivo' }));
    expect(errors.getByRole('row', { name: /5 precio Fila 5, columna «precio»: no es un número/ })).toBeInTheDocument();
    expect(errors.getByRole('row', { name: /9 unidad .*«bulto» no existe/ })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Importar' })).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Subir otro archivo' })).toBeInTheDocument();
  });

  it('subir otro archivo vuelve al formulario', async () => {
    importApi({ 'POST /api/v1/imports': json(201, aRejectedPreview()) });
    await upload();

    await userEvent.click(await screen.findByRole('button', { name: 'Subir otro archivo' }));

    expect(await screen.findByRole('button', { name: 'Revisar archivo' })).toBeInTheDocument();
  });

  it('confirmar pone la importación en cola y lleva a su avance', async () => {
    const api = importApi();
    const router = await upload();

    await screen.findByRole('table', { name: 'Primeros productos del archivo' });
    await confirm();

    await waitFor(() => expect(router.state.location.pathname).toBe(`/productos/importar/${importId}`));
    expect(api.calls(`POST /api/v1/imports/${importId}/confirm`)).toHaveLength(1);
  });
});
