import { formatQuantity, isNegative } from '@/shared/format';
import { useVariantStock } from '../../api/stock';
import { balanceAfter } from '../../model/stockChange/stockChange';
import styles from './BalanceChange.module.scss';

type Props = {
  variantId: string;
  locationId: string;
  locationName: string;
  unit: string;
  /** The signed change. */
  delta: string;
};

/** What the movement would leave in a location. Only a preview: the API decides, so it says nothing until it can read. */
export function BalanceChange({ variantId, locationId, locationName, unit, delta }: Props) {
  const { data: current } = useVariantStock(variantId, locationId);
  if (current === undefined) return null;
  const after = balanceAfter(current, delta);
  const negative = isNegative(after);
  return (
    <p className={styles.root} data-tone={negative ? 'danger' : undefined}>
      {locationName} pasa de {formatQuantity(current)} a {formatQuantity(after)} {unit}.
      {negative && ' Quedaría negativa.'}
    </p>
  );
}
