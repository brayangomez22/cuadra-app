import { Field, Input, Select } from '@/shared/ui';
import { useCategories } from '../../../../api/categories';
import type { ProductStatus } from '../../../../api/types';
import type { ProductListParams } from '../../hooks/useProductListParams';
import styles from './ProductFilters.module.scss';

const statusOptions: { value: ProductStatus; label: string }[] = [
  { value: 'active', label: 'Activos' },
  { value: 'inactive', label: 'Inactivos' },
  { value: 'all', label: 'Todos' },
];
const ALL = 'todas';

const isStatus = (value: string): value is ProductStatus => statusOptions.some((option) => option.value === value);

type Props = { params: ProductListParams; canSeeMargin: boolean };

/** The search by name, SKU or barcode, the status, the category and the order of the product list. */
export function ProductFilters({ params, canSeeMargin }: Props) {
  const categories = useCategories();
  const sorts = [
    { value: 'name', label: 'Nombre' },
    { value: 'price', label: 'Precio, menor primero' },
    ...(canSeeMargin ? [{ value: 'margin', label: 'Margen, menor primero' }] : []),
  ];
  return (
    <div role="search" className={styles.root}>
      <Field
        label="Buscar"
        className={styles.search}
        hint={params.q ? 'Con búsqueda, primero lo que más coincide.' : undefined}
      >
        <Input
          type="search"
          value={params.text}
          onChange={(event) => params.setText(event.target.value)}
          placeholder="Nombre, SKU o código de barras"
          autoComplete="off"
          enterKeyHint="search"
        />
      </Field>
      <Field label="Estado">
        <Select
          options={statusOptions}
          value={params.status}
          onValueChange={(value) => isStatus(value) && params.setStatus(value)}
        />
      </Field>
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
      <Field label="Ordenar por">
        <Select options={sorts} value={params.sort} onValueChange={params.setSort} />
      </Field>
    </div>
  );
}
