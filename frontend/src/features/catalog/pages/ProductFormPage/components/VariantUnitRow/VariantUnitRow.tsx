import { formatMoney, formatQuantity } from '@/shared/format';
import { IconButton, IconEdit, IconTrash } from '@/shared/ui';
import { useRemoveVariantUnit } from '../../../../api/variantUnits';
import type { Product, ProductVariant, VariantUnit } from '../../../../api/types';
import styles from './VariantUnitRow.module.scss';

type Props = { product: Product; variant: ProductVariant; unit: VariantUnit; onEdit: () => void };

/** An alternate unit of a variant: its equivalence in the base unit, its prices and its barcode. */
export function VariantUnitRow({ product, variant, unit, onEdit }: Props) {
  const remove = useRemoveVariantUnit();
  const { code } = unit.unit;
  return (
    <li className={styles.root} aria-label={code}>
      <div className={styles.text}>
        <span className={styles.equivalence}>
          1 {code} = {formatQuantity(unit.factor)} {product.unit.code}
        </span>
        <span className={styles.meta}>
          {formatMoney(unit.price)} sin IVA · {formatMoney(unit.price_with_tax)} con IVA
        </span>
        {unit.barcode && <span className={styles.barcode}>{unit.barcode}</span>}
      </div>
      <div className={styles.actions}>
        <IconButton label={`Editar ${code}`} icon={<IconEdit />} onClick={onEdit} />
        <IconButton
          label={`Quitar ${code}`}
          icon={<IconTrash />}
          variant="danger"
          onClick={() => remove.mutate({ productId: product.id, variantId: variant.id, unit: unit.unit })}
        />
      </div>
    </li>
  );
}
