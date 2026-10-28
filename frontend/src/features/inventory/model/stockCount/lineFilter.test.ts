import { lineFilterOptions, lineFilterParam, toLineStatus } from './lineFilter';

describe('lineFilter', () => {
  it.each([
    ['revision', 'review'],
    ['diferencia', 'difference'],
    ['sin-contar', 'uncounted'],
    ['exactos', 'exact'],
    ['todas', undefined],
  ] as const)('?estado=%s pide a la API las líneas %s', (filter, status) => {
    expect(toLineStatus(filter)).toBe(status);
  });

  it('un estado desconocido en la URL muestra todas las líneas', () => {
    expect(lineFilterParam.parse('otra-cosa')).toBeUndefined();
    expect(lineFilterParam.fallback).toBe('todas');
  });

  it('ofrece todas las líneas primero', () => {
    expect(lineFilterOptions.map((o) => o.label)).toEqual([
      'Todas',
      'En revisión',
      'Con diferencia',
      'Sin contar',
      'Exactas',
    ]);
  });
});
