import { anImport, counts } from '../testing/importApi';
import { previewSummary } from './previewSummary';

describe('previewSummary', () => {
  it('un archivo válido de una ferretería se puede importar', () => {
    const summary = previewSummary(anImport({ counts: counts({ rows: 8001, products: 8000 }) }));

    expect(summary).toEqual({
      title: 'El archivo está listo',
      text: '8.000 productos en 8.001 filas. Revise los primeros y confirme para importarlo.',
    });
  });

  it('un archivo de un almacén de ropa con un error no importa nada', () => {
    const summary = previewSummary(
      anImport({ status: 'rejected', counts: counts({ rows: 1, products: 1, errors: 1 }) }),
    );

    expect(summary).toEqual({
      title: 'El archivo tiene 1 error',
      text: '1 producto en 1 fila. No se importó nada: corrija el archivo y súbalo de nuevo.',
    });
  });
});
