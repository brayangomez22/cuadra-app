import type { RefObject } from 'react';
import type { Me } from '@/features/auth';
import { formatPercent, isZero } from '@/shared/format';
import { cartItem } from '../../../../model/cart/cartItem';
import type { DiscountLimit } from '../../../../model/cart/discount';
import { useAddToCart } from '../../hooks/useAddToCart';
import type { CartState } from '../../hooks/useCart';
import { CartLines } from '../CartLines/CartLines';
import { ProductSearch } from '../ProductSearch/ProductSearch';
import styles from './SaleCart.module.scss';

type Props = {
  cart: CartState;
  limit: DiscountLimit;
  searchRef: RefObject<HTMLInputElement | null>;
  /** Gives the focus back to the search, for the next product. */
  onDone: () => void;
  user: Me | undefined;
};

/** The search and the lines of the sale. */
export function SaleCart({ cart, limit, searchRef, onDone, user }: Props) {
  const { added, announcement, add } = useAddToCart({ cart: cart.cart, change: cart.change, limit });
  const showDiscount = limit.unlimited || !isZero(limit.max);
  return (
    <div className={styles.root}>
      <ProductSearch
        ref={searchRef}
        // A unit's barcode sells that unit, with the module on or off: the API
        // sells the units already defined, and a box charged as one unit
        // would be a wrong charge (docs/decisiones.md, T26a).
        onAdd={({ product, variant, unit }, quantity) =>
          add({ type: 'add', item: cartItem(product, variant), quantity, unitId: unit?.unit.id })
        }
      />
      <p className="srOnly" role="status" aria-label="Último producto agregado">
        {announcement}
      </p>
      {showDiscount && !limit.unlimited && (
        <p className={styles.note}>Descuento máximo por línea: {formatPercent(limit.max, 'rate')}</p>
      )}
      <CartLines
        totals={cart.totals}
        added={added}
        alternateUnits={user?.modules.includes('unit_conversions') ?? false}
        dispatch={cart.change}
        showDiscount={showDiscount}
        outOfStock={cart.outOfStock}
        onDone={onDone}
      />
    </div>
  );
}
