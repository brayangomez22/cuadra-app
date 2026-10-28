import { importSchema, MAX_FILE_BYTES } from './importSchema';

const values = (file: File | null) => ({ location_id: 'sede', on_existing_sku: 'update' as const, file });
const fileOf = (name: string, size = 10) => Object.defineProperty(new File(['x'], name), 'size', { value: size });

describe('importSchema', () => {
  it.each([
    ['un .xlsx de una ferretería', fileOf('ferreteria.xlsx')],
    ['un .csv de un almacén de ropa, con la extensión en mayúsculas', fileOf('ROPA.CSV')],
    ['un archivo de 10 MB justos', fileOf('catalogo.xlsx', MAX_FILE_BYTES)],
  ])('acepta %s', (_, file) => {
    expect(importSchema.safeParse(values(file)).success).toBe(true);
  });

  it.each([
    ['sin archivo', null, 'Elija el archivo'],
    ['un .pdf', fileOf('catalogo.pdf'), 'El archivo debe ser .xlsx o .csv'],
    ['un .xls viejo', fileOf('catalogo.xls'), 'El archivo debe ser .xlsx o .csv'],
    ['un archivo de más de 10 MB', fileOf('catalogo.csv', MAX_FILE_BYTES + 1), 'El archivo pasa de 10 MB'],
  ])('rechaza %s', (_, file, message) => {
    const result = importSchema.safeParse(values(file));

    expect(result.error?.issues.map((issue) => issue.message)).toEqual([message]);
  });

  it('exige la sede', () => {
    const result = importSchema.safeParse({ ...values(fileOf('a.csv')), location_id: '' });

    expect(result.error?.issues.map((issue) => issue.message)).toEqual(['Elija la sede']);
  });
});
