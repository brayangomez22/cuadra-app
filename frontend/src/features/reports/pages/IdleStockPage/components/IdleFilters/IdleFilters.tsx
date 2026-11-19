import { useCategories } from '@/features/catalog';
import { Field, Select } from '@/shared/ui';
import { IDLE_DAYS, type IdleParams } from '../../hooks/useIdleParams';

const ALL = 'todas';
const dayOptions = IDLE_DAYS.map((d) => ({ value: d, label: `${d} días` }));

/** How long without selling, and the category. */
export function IdleFilters({ params }: { params: IdleParams }) {
  const categories = useCategories();
  return (
    <>
      <Field label="Sin vender hace">
        <Select options={dayOptions} value={params.days} onValueChange={params.setDays} />
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
    </>
  );
}
