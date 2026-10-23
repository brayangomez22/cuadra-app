import { useEffect, useRef, type Dispatch } from 'react';
import { EmptyState, IconCart } from '@/shared/ui';
import { keyOf } from '../../../../model/cart/cartLine';
import type { CartAction } from '../../../../model/cart/cartReducer';
import type { CartTotals } from '../../../../model/cart/cartTotals';
import type { AddedMark } from '../../hooks/useAddedMark';
import { CartLine } from '../CartLine/CartLine';
import styles from './CartLines.module.scss';

type Props = {
  totals: CartTotals;
  dispatch: Dispatch<CartAction>;
  /** Whether the user may give discounts: without a limit above 0 %, the field is not shown. */
  showDiscount: boolean;
  /** The variant the API said has no stock for the sale. */
  outOfStock?: string;
  /** The line just added: it is marked and scrolled into view. */
  added?: AddedMark;
  /** Whether the module unit_conversions is on: a line of a variant with alternate units picks its unit. */
  alternateUnits: boolean;
  /** Gives the focus back to the search, for the next product. */
  onDone: () => void;
};

/**
 * The lines of the sale in progress, the newest last. Quantity and discount
 * are edited in place; Enter in either goes back to the search.
 */
export function CartLines({ totals, added, ...line }: Props) {
  const lines = useRef(new Map<string, HTMLLIElement>());

  // On a phone or at 200% zoom the added line can be off screen: bring it in,
  // without moving the page if it is already visible. The focus stays in the search.
  const addedSeq = added?.seq;
  const addedId = added?.key;
  useEffect(() => {
    if (addedId) lines.current.get(addedId)?.scrollIntoView({ block: 'nearest' });
  }, [addedId, addedSeq]);

  if (totals.lines.length === 0) {
    return (
      <EmptyState
        className={styles.empty}
        icon={<IconCart />}
        title="Escanee o busque un producto"
        description="Los productos de la venta aparecen aquí."
      />
    );
  }

  return (
    <ol className={styles.root} aria-label="Productos de la venta">
      {totals.lines.map((totalsOfLine) => {
        const key = keyOf(totalsOfLine.line);
        return (
          <CartLine
            key={key}
            ref={(element) => {
              if (element) lines.current.set(key, element);
              else lines.current.delete(key);
            }}
            totals={totalsOfLine}
            added={key === added?.key}
            {...line}
          />
        );
      })}
    </ol>
  );
}
