import { useFormContext, useWatch } from 'react-hook-form';
import { PurchaseTotals } from '../../../../components/PurchaseTotals/PurchaseTotals';
import type { PurchaseInputValues } from '../../../../model/purchaseForm/purchaseSchema';
import { purchaseTotals } from '../../../../model/purchaseLines/purchaseTotals';

/** The amounts of the purchase as the lines are typed; the API has the last word when it is registered. */
export function LiveTotals() {
  const { control } = useFormContext<PurchaseInputValues>();
  const lines = useWatch({ control, name: 'lines' });
  return <PurchaseTotals {...purchaseTotals(lines)} />;
}
