import { FormSection } from '@/shared/ui';
import { useState } from 'react';
import type { Product, ProductVariant, UnitOfMeasure, VariantUnit } from '../../../../api/types';
import { UnitDialog } from '../UnitDialog/UnitDialog';
import { VariantUnitList } from '../VariantUnitList/VariantUnitList';
import styles from './UnitsSection.module.scss';

type Props = {
  /** The saved product; a new one has no variants to add units to yet. */
  product?: Product;
  /** The tenant's units. */
  units: UnitOfMeasure[];
};

type Editing = { variant: ProductVariant; unit?: VariantUnit };

/** "Unidades de venta" (module unit_conversions): for each variant, its alternate units. */
export function UnitsSection({ product, units }: Props) {
  const [editing, setEditing] = useState<Editing | null>(null);
  if (!product) {
    return (
      <FormSection title="Unidades de venta">
        <p className={styles.hint}>Guarde el producto para agregar las otras unidades en que se vende.</p>
      </FormSection>
    );
  }
  return (
    <FormSection title="Unidades de venta">
      <p className={styles.hint}>
        Otras presentaciones en que se vende y se compra, cada una con su precio. El inventario se lleva en{' '}
        {product.unit.code}.
      </p>
      <div className={styles.root}>
        {product.variants.map((variant) => (
          <VariantUnitList
            key={variant.id}
            product={product}
            variant={variant}
            onEdit={(unit) => setEditing({ variant, unit })}
          />
        ))}
      </div>
      {editing && (
        <UnitDialog
          key={`${editing.variant.id}-${editing.unit?.unit.id ?? 'new'}`}
          target={{ product, variant: editing.variant, unit: editing.unit }}
          units={units}
          onClose={() => setEditing(null)}
        />
      )}
    </FormSection>
  );
}
