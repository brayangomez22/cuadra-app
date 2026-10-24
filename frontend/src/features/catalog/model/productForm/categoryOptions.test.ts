import { categoryOptions } from './categoryOptions';
import { NO_CATEGORY } from './productSchema';

const category = (id: string, name: string, parent_id?: string) => ({
  id,
  name,
  created_at: '2026-10-01T15:00:00Z',
  ...(parent_id && { parent_id }),
});

describe('categoryOptions', () => {
  it('ordena por nombre, una subcategoría muestra su ruta y Sin categoría va primero', () => {
    const categories = [
      category('c2', 'Plomería'),
      category('c3', 'Tubería', 'c2'),
      category('c1', 'Camisetas', 'c4'),
      category('c4', 'Ropa'),
    ];

    expect(categoryOptions(categories)).toEqual([
      { value: NO_CATEGORY, label: 'Sin categoría' },
      { value: 'c2', label: 'Plomería' },
      { value: 'c3', label: 'Plomería › Tubería' },
      { value: 'c4', label: 'Ropa' },
      { value: 'c1', label: 'Ropa › Camisetas' },
    ]);
  });

  it('una categoría cuyo padre no está se muestra sola', () => {
    expect(categoryOptions([category('c1', 'Analgésicos', 'perdida')])).toEqual([
      { value: NO_CATEGORY, label: 'Sin categoría' },
      { value: 'c1', label: 'Analgésicos' },
    ]);
  });
});
