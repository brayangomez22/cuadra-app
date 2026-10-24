import { useFormContext } from 'react-hook-form';
import { Alert } from '@/shared/ui';
import type { VariantAxis } from '../../../../api/types';
import type { ProductFormInput } from '../../../../model/productForm/productSchema';
import { MAX_COMBINATIONS } from '../../../../model/variantMatrix/variantChoice';
import { useVariantChoice } from '../../hooks/useVariantChoice';
import { AxisPicker } from '../AxisPicker/AxisPicker';
import { VariantMatrix } from '../VariantMatrix/VariantMatrix';
import styles from './VariantEditor.module.scss';

type Props = {
  axes: VariantAxis[];
  /** An inactive product's prices cannot change. */
  inactive: boolean;
};

/**
 * The variants of a product (module variants): the person picks values of
 * the tenant's axes and each combination is a variant with its own SKU,
 * barcode, cost and price. Empty fields take the product's. With no value
 * picked, the product is simple.
 */
export function VariantEditor({ axes, inactive }: Props) {
  const { control } = useFormContext<ProductFormInput>();
  const choice = useVariantChoice(control, axes);
  const { lockedAxes } = choice;
  return (
    <div className={styles.root}>
      <div className={styles.axes}>
        {choice.allAxes.map((axis) => (
          <AxisPicker
            key={axis.name}
            axis={axis}
            chosen={choice.chosen[axis.name] ?? []}
            locked={choice.isAxisLocked(axis.name)}
            isValueUsed={(value) => choice.isValueUsed(axis.name, value)}
            onToggle={(value) => choice.toggle(axis.name, value)}
          />
        ))}
      </div>
      {lockedAxes && lockedAxes.length > 0 && (
        <p className={styles.hint}>
          Sus variantes varían en {lockedAxes.join(' y ')}: puede agregar valores, pero no quitar los que ya tienen
          variantes.
        </p>
      )}
      {choice.tooMany && (
        <Alert tone="warning">
          Son más de {MAX_COMBINATIONS} variantes. Elija menos valores y agregue el resto después.
        </Alert>
      )}
      {choice.fields.length > 0 && <VariantMatrix fields={choice.fields} rows={choice.rows} inactive={inactive} />}
    </div>
  );
}
