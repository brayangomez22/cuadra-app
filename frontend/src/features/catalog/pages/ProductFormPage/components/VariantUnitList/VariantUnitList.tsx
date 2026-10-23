import { Button, IconPlus } from '@/shared/ui';
import type { Product, ProductVariant, VariantUnit } from '../../../../api/types';
import { hasAxes } from '../../../../model/variants';
import { VariantUnitRow } from '../VariantUnitRow/VariantUnitRow';
import styles from './VariantUnitList.module.scss';

type Props = {
  product: Product;
  variant: ProductVariant;
  /** Opens the dialog for a unit of the variant, or for a new one. */
  onEdit: (unit?: VariantUnit) => void;
};

/** The alternate units of one variant and the button to add one. A product with axes names the variant. */
export function VariantUnitList({ product, variant, onEdit }: Props) {
  const byAxes = hasAxes(product);
  const label = byAxes ? variant.label : '';
  const base = product.unit.code;
  return (
    <div className={styles.root} role={byAxes ? 'group' : undefined} aria-label={label || undefined}>
      {byAxes && <span className={styles.label}>{label}</span>}
      {variant.units.length === 0 ? (
        <p className={styles.hint}>Sin unidades de venta: se vende solo por {base}.</p>
      ) : (
        <ul className={styles.list}>
          {variant.units.map((unit) => (
            <VariantUnitRow
              key={unit.unit.id}
              product={product}
              variant={variant}
              unit={unit}
              onEdit={() => onEdit(unit)}
            />
          ))}
        </ul>
      )}
      <Button
        variant="secondary"
        icon={<IconPlus />}
        className={styles.add}
        aria-label={byAxes ? `Agregar unidad a ${label}` : undefined}
        onClick={() => onEdit()}
      >
        Agregar unidad
      </Button>
    </div>
  );
}
