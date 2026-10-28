import { plain } from '@/test/fixtures';
import { sample } from '../testing/importApi';
import { sampleRows } from './sampleRows';

describe('sampleRows', () => {
  it('una fila por variante, con los valores de sus ejes, de una ferretería y un almacén de ropa', () => {
    const rows = sampleRows(sample);

    expect(rows.map(({ row, name, sku, variant }) => ({ row, name, sku, variant }))).toEqual([
      { row: 2, name: 'Martillo de uña 16 oz', sku: 'MART-16', variant: '' },
      { row: 3, name: 'Camiseta básica', sku: 'CAMB-M', variant: 'talla: M' },
      { row: 4, name: 'Camiseta básica', sku: 'CAMB-L', variant: 'talla: L' },
    ]);
    expect(plain(rows[0]?.price)).toBe('$ 25.000');
    expect(rows[0]?.stock).toBe('12');
    expect(rows[2]?.stock).toBe('');
  });

  it('un precio que no es un número se muestra como vino en el archivo', () => {
    const [product] = sample;
    const variant = { ...product!.variants[0]!, price: 'veinte mil' };

    expect(sampleRows([{ ...product!, variants: [variant] }])[0]?.price).toBe('veinte mil');
  });

  it('un precio vacío en el archivo se muestra vacío, no como $ 0', () => {
    const [product] = sample;
    const variant = { ...product!.variants[0]!, price: '' };

    expect(sampleRows([{ ...product!, variants: [variant] }])[0]?.price).toBe('');
  });
});
