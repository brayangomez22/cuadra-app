import { hasAxes, type Product, type ProductVariant } from '@/features/catalog';
import { Field, Select } from '@/shared/ui';
import type { Location } from '../../../../api/types';
import type { useKardexFilters } from '../../hooks/useKardexFilters';
import styles from './KardexFilters.module.scss';

const ALL_LOCATIONS = 'all';

type Props = {
  product: Product;
  variant: ProductVariant;
  locations: { data?: Location[]; isPending: boolean };
  filters: ReturnType<typeof useKardexFilters>;
};

/** The variant, only when the product has axes, and the location (or every location). */
export function KardexFilters({ product, variant, locations, filters }: Props) {
  const locationOptions = [
    { value: ALL_LOCATIONS, label: 'Todas las sedes' },
    ...(locations.data ?? []).map((l) => ({ value: l.id, label: l.name })),
  ];
  return (
    <div className={styles.root}>
      {hasAxes(product) && (
        <Field label="Variante" className={styles.filter}>
          <Select
            options={product.variants.map((v) => ({ value: v.id, label: v.label || v.sku }))}
            value={variant.id}
            onValueChange={filters.chooseVariant}
          />
        </Field>
      )}
      <Field label="Sede" className={styles.filter}>
        <Select
          options={locationOptions}
          value={filters.locationId ?? ALL_LOCATIONS}
          onValueChange={(value) => filters.chooseLocation(value === ALL_LOCATIONS ? undefined : value)}
          disabled={locations.isPending}
        />
      </Field>
    </div>
  );
}
