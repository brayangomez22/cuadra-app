import { useCategories } from '@/features/catalog';
import { Field, Select } from '@/shared/ui';
import type { ProductsParams } from '../../hooks/useProductsParams';
import { AttributeFilter } from '../AttributeFilter/AttributeFilter';

// The value of "every category" in the select.
const ALL = 'todas';

type Props = { params: ProductsParams; seesCosts: boolean };

/** The category, a tenant attribute and its value, and the order of the products report. */
export function ProductFilters({ params, seesCosts }: Props) {
  const categories = useCategories();
  const sorts = [
    { value: 'revenue', label: 'Más ingreso' },
    { value: 'quantity', label: 'Más cantidad' },
    ...(seesCosts ? [{ value: 'profit', label: 'Más utilidad' }] : []),
  ];
  return (
    <>
      <Field label="Categoría">
        <Select
          options={[
            { value: ALL, label: 'Todas' },
            ...(categories.data ?? []).map((c) => ({ value: c.id, label: c.name })),
          ]}
          value={params.category || ALL}
          onValueChange={(value) => params.setCategory(value === ALL ? '' : value)}
        />
      </Field>
      <AttributeFilter params={params} />
      <Field label="Ordenar por">
        <Select options={sorts} value={params.sort} onValueChange={params.setSort} />
      </Field>
    </>
  );
}
