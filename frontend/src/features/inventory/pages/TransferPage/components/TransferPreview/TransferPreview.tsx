import { useFormContext, useWatch } from 'react-hook-form';
import { BalanceChange } from '../../../../components/BalanceChange/BalanceChange';
import { typedQuantity } from '../../../../model/stockChange/stockChange';
import type { TransferFormInput } from '../../../../model/stockChange/transferSchema';

/** What each location would have after the transfer, once the form describes one. */
export function TransferPreview({ nameOf }: { nameOf: (locationId: string) => string }) {
  const { control } = useFormContext<TransferFormInput>();
  const [item, from, to, text] = useWatch({
    control,
    name: ['item', 'from_location_id', 'to_location_id', 'quantity'],
  });
  const quantity = typedQuantity(text);
  if (!item || !quantity || from === to) return null;
  const change = { variantId: item.variant.id, unit: item.product.unit.code };
  return (
    <>
      {from && <BalanceChange {...change} locationId={from} locationName={nameOf(from)} delta={`-${quantity}`} />}
      {to && <BalanceChange {...change} locationId={to} locationName={nameOf(to)} delta={quantity} />}
    </>
  );
}
