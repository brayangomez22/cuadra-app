import { ApiError } from '@/shared/api/errors';
import type { ResolvedRow } from './resolveRows';
import { takenCodeRow } from './takenCodeRow';

const row = (sku: string, barcode = ''): ResolvedRow => ({
  values: {},
  sku,
  barcode,
  cost: '1000',
  price: '2000',
  active: true,
});

const rows = [row('CAM-S'), row('CAM-M', '7700000000002'), row('TOR-CAJA', '7701111111111')];

const taken = (code: string, details: Record<string, string>) =>
  new ApiError(409, { error: { code, message: 'Ya existe', details } });

describe('takenCodeRow', () => {
  it.each([
    { name: 'el SKU de una prenda', error: taken('sku_taken', { sku: 'CAM-M' }), index: 1, field: 'sku' },
    {
      name: 'el código de barras de una caja',
      error: taken('barcode_taken', { barcode: '7701111111111' }),
      index: 2,
      field: 'barcode',
    },
  ])('marca la fila con $name', ({ error, index, field }) => {
    expect(takenCodeRow(error, rows)).toEqual({ index, field, message: 'Ya existe' });
  });

  it.each([
    { name: 'sin details', error: taken('sku_taken', {}) },
    { name: 'un código que no está en las filas', error: taken('barcode_taken', { barcode: '7709999999999' }) },
    { name: 'otro código de error', error: taken('invalid_price', { sku: 'CAM-M' }) },
    { name: 'un error de red', error: new TypeError('Failed to fetch') },
  ])('no marca ninguna fila con $name', ({ error }) => {
    expect(takenCodeRow(error, rows)).toBeUndefined();
  });
});
