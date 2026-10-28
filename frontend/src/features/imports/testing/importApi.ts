import { aUser, fakeApi, json, signedIn, type Me } from '@/test/fakeApi';
import { aLocation, ids } from '@/test/fixtures';
import type { Import, ImportCounts, ImportPreview, ImportProductPreview, ImportRowError } from '../api/types';

export const importId = '01920000-0000-7000-8000-000000000a01';

export function counts(overrides: Partial<ImportCounts> = {}): ImportCounts {
  return { rows: 3, products: 2, created: 0, updated: 0, skipped: 0, errors: 0, ...overrides };
}

export function anImport(overrides: Partial<Import> = {}): Import {
  return {
    id: importId,
    file_name: 'catalogo.xlsx',
    format: 'xlsx',
    status: 'validated',
    on_existing_sku: 'update',
    location_id: ids.main,
    counts: counts(),
    created_at: '2026-10-27T15:00:00Z',
    started_at: null,
    finished_at: null,
    ...overrides,
  };
}

export function aRowError(row: number, column: string, message: string): ImportRowError {
  return { row, column, code: 'invalid_value', message };
}

// One product of a hardware store and one garment with two sizes: the sample of every preview.
export const sample: ImportProductPreview[] = [
  {
    row: 2,
    name: 'Martillo de uña 16 oz',
    category: 'Herramientas',
    unit: 'Unidad',
    variants: [{ row: 2, sku: 'MART-16', price: '25000', axis_values: {}, initial_stock: '12' }],
  },
  {
    row: 3,
    name: 'Camiseta básica',
    category: 'Camisetas',
    unit: 'Unidad',
    variants: [
      { row: 3, sku: 'CAMB-M', price: '30000', axis_values: { talla: 'M' }, initial_stock: '5' },
      { row: 4, sku: 'CAMB-L', price: '32000', axis_values: { talla: 'L' }, initial_stock: null },
    ],
  },
];

export function aPreview(overrides: Partial<ImportPreview> = {}): ImportPreview {
  return { import: anImport(), errors: [], sample, ...overrides };
}

export const rejectedErrors = [
  aRowError(5, 'precio', 'Fila 5, columna «precio»: no es un número.'),
  aRowError(9, 'unidad', 'Fila 9, columna «unidad»: la unidad «bulto» no existe.'),
];

export function aRejectedPreview(): ImportPreview {
  const rejected = anImport({ status: 'rejected', counts: counts({ errors: 2 }) });
  return aPreview({ import: rejected, errors: rejectedErrors, sample: sample.slice(0, 1) });
}

/** The API of the import screens: two locations, a valid upload and its confirmation. */
export function importApi(extra: Parameters<typeof fakeApi>[0] = {}, user: Me = aUser()) {
  return fakeApi({
    ...signedIn(user),
    'GET /api/v1/locations': json(200, {
      items: [aLocation(), aLocation({ id: ids.north, name: 'Sede norte' })],
    }),
    'POST /api/v1/imports': json(201, aPreview()),
    'POST /api/v1/imports/{importId}/confirm': json(200, anImport({ status: 'queued' })),
    'GET /api/v1/imports/{importId}/errors': json(200, { items: rejectedErrors, total: 2 }),
    ...extra,
  });
}
