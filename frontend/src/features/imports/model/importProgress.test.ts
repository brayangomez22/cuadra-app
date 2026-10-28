import type { ImportStatus } from '../api/types';
import { counts } from '../testing/importApi';
import { importStage, processed, resultLines } from './importProgress';

describe('importStage', () => {
  it.each<[ImportStatus, ReturnType<typeof importStage>]>([
    ['queued', 'running'],
    ['running', 'running'],
    ['completed', 'completed'],
    ['failed', 'failed'],
    ['rejected', 'unconfirmed'],
    ['validated', 'unconfirmed'],
  ])('%s se muestra como %s', (status, stage) => {
    expect(importStage(status)).toBe(stage);
  });
});

describe('processed', () => {
  it('cuenta los productos creados, actualizados y omitidos', () => {
    expect(processed(counts({ products: 10, created: 3, updated: 2, skipped: 1 }))).toBe(6);
  });
});

describe('resultLines', () => {
  it.each([
    ['una ferretería que carga su catálogo', counts({ created: 8000 }), ['8.000 creados']],
    [
      'un almacén de ropa que repite el archivo',
      counts({ created: 1, updated: 2, skipped: 1 }),
      ['1 creado', '2 actualizados', '1 omitido'],
    ],
    ['un archivo sin productos nuevos', counts({ updated: 1 }), ['1 actualizado']],
  ])('%s', (_, given, lines) => {
    expect(resultLines(given)).toEqual(lines);
  });
});
