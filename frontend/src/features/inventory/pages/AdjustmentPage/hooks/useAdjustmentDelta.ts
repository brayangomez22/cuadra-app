import { useWatch, type Control } from 'react-hook-form';
import { formatQuantity, isZero } from '@/shared/format';
import { useVariantStock } from '../../../api/stock';
import type { AdjustmentFormInput, AdjustmentValues } from '../../../model/stockChange/adjustmentSchema';
import { countDelta, signedQuantity } from '../../../model/stockChange/stockChange';

/**
 * The stock of the chosen product in the chosen location, and the change the
 * submitted values make: what leaves or enters, or for a count what was
 * counted less what there is. A change of zero, or a count before the stock is
 * known, is not an adjustment: then it says why instead.
 */
export function useAdjustmentDelta(control: Control<AdjustmentFormInput, unknown, AdjustmentValues>) {
  const [item, locationId] = useWatch({ control, name: ['item', 'location_id'] });
  const current = useVariantStock(item?.variant.id, locationId || undefined).data;

  const deltaOf = ({ item: chosen, direction, quantity }: AdjustmentValues): { delta: string } | { error: string } => {
    if (direction !== 'count') return { delta: signedQuantity(direction as 'out' | 'in', quantity) };
    if (current === undefined) return { error: 'Aún no sabemos cuánto hay; intente de nuevo en un momento.' };
    const delta = countDelta(quantity, current);
    if (!isZero(delta)) return { delta };
    return { error: `Ya hay ${formatQuantity(current)} ${chosen.product.unit.code}: no hay nada que ajustar.` };
  };

  return { current, deltaOf };
}
