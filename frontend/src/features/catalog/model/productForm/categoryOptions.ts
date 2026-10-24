import type { SelectOption } from '@/shared/ui';
import type { Category } from '../../api/types';
import { NO_CATEGORY } from './productSchema';

// Deeper than this, a broken chain of parents stops instead of looping.
const MAX_DEPTH = 10;

/** The categories to choose from, by name: a subcategory shows its path ("Categoría › Subcategoría"). */
export function categoryOptions(categories: Category[]): SelectOption[] {
  const byId = new Map(categories.map((category) => [category.id, category]));
  const path = (category: Category, depth = 0): string => {
    const parent = category.parent_id ? byId.get(category.parent_id) : undefined;
    return parent && depth < MAX_DEPTH ? `${path(parent, depth + 1)} › ${category.name}` : category.name;
  };
  const options = categories.map((category) => ({ value: category.id, label: path(category) }));
  options.sort((a, b) => a.label.localeCompare(b.label, 'es'));
  return [{ value: NO_CATEGORY, label: 'Sin categoría' }, ...options];
}
