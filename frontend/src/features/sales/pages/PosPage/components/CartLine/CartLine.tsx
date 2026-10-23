import type { Dispatch, Ref } from 'react';
import { variantName } from '@/features/catalog';
import { formatMoney, isZero } from '@/shared/format';
import { Badge, IconAlert, IconButton, IconClose } from '@/shared/ui';
import { lineUnit, type LineRef } from '../../../../model/cart/cartLine';
import type { CartAction } from '../../../../model/cart/cartReducer';
import type { LineAmounts, LineTotals } from '../../../../model/cart/lineTotals';
import { LineFields } from '../LineFields/LineFields';
import { UnitSelect } from '../UnitSelect/UnitSelect';
import styles from './CartLine.module.scss';

type Props = {
  totals: LineTotals;
  dispatch: Dispatch<CartAction>;
  showDiscount: boolean;
  outOfStock?: string;
  /** Whether it is the line just added. */
  added: boolean;
  alternateUnits: boolean;
  onDone: () => void;
  ref: Ref<HTMLLIElement>;
};

/** A line of the sale: the variant, its unit, quantity, discount and amount. */
export function CartLine({ totals, dispatch, showDiscount, outOfStock, added, alternateUnits, onDone, ref }: Props) {
  const { line } = totals;
  const { item } = line;
  const lineRef: LineRef = { variantId: item.variantId, unitId: line.unitId };
  // A variant in two units is two lines: the alternate one says its unit.
  const product = variantName(item.name, item.label);
  const name = line.unitId ? `${product}, ${lineUnit(line).code}` : product;
  const forProduct = <span className="srOnly">de {name}</span>;
  const noStock = item.variantId === outOfStock;
  const fields = { totals, lineRef, dispatch, forProduct, onDone };

  return (
    <li ref={ref} className={styles.root} aria-label={name} data-state={lineState({ noStock, added })}>
      <div className={styles.product}>
        <span className={styles.name}>{product}</span>
        <span className={styles.meta}>
          {item.sku} · {formatMoney(lineUnit(line).priceWithTax)} / {lineUnit(line).code}
        </span>
        {alternateUnits && item.units.length > 0 && (
          <UnitSelect line={line} label={<>Unidad {forProduct}</>} dispatch={dispatch} onDone={onDone} />
        )}
        {noStock && (
          <Badge tone="danger" icon={<IconAlert />}>
            Sin stock suficiente
          </Badge>
        )}
      </div>
      <LineFields {...fields} showDiscount={showDiscount} />
      <LineAmount amounts={totals.amounts} />
      <IconButton
        className={styles.remove}
        label={`Quitar ${name}`}
        icon={<IconClose />}
        onClick={() => {
          dispatch({ type: 'remove', ...lineRef });
          onDone();
        }}
      />
    </li>
  );
}

function LineAmount({ amounts }: { amounts: LineAmounts | undefined }) {
  return (
    <div className={styles.amount}>
      {amounts ? (
        <span className={styles.total}>{formatMoney(amounts.total)}</span>
      ) : (
        <span className={styles.meta}>Sin total</span>
      )}
      {amounts && !isZero(amounts.discount) && (
        <span className={styles.meta}>−{formatMoney(amounts.discount)} antes de IVA</span>
      )}
    </div>
  );
}

function lineState({ noStock, added }: { noStock: boolean; added: boolean }) {
  if (noStock) return 'error';
  return added ? 'added' : undefined;
}
