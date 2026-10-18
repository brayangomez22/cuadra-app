import { useEffect, useRef, type Dispatch, type KeyboardEvent } from "react";
import { variantName } from "@/features/catalog";
import { formatMoney, isZero } from "@/shared/format";
import {
  Badge,
  EmptyState,
  Field,
  IconAlert,
  IconButton,
  IconCart,
  IconClose,
  Input,
} from "@/shared/ui";
import type { CartAction, CartTotals } from "./cart";
import styles from "./CartLines.module.scss";

type Props = {
  totals: CartTotals;
  dispatch: Dispatch<CartAction>;
  /** Whether the user may give discounts: without a limit above 0 %, the field is not shown. */
  showDiscount: boolean;
  /** The product the API said has no stock for the sale. */
  outOfStock?: string;
  /** The variant just added: its line is marked and scrolled into view. seq changes on every add. */
  added?: { variantId: string; seq: number };
  /** Gives the focus back to the search, for the next product. */
  onDone: () => void;
};

/**
 * The lines of the sale in progress, the newest last. Quantity and discount
 * are edited in place; Enter in either goes back to the search.
 */
export function CartLines({
  totals,
  dispatch,
  showDiscount,
  outOfStock,
  added,
  onDone,
}: Props) {
  const lines = useRef(new Map<string, HTMLLIElement>());

  // On a phone or at 200% zoom the added line can be off screen: bring it in,
  // without moving the page if it is already visible. The focus stays in the search.
  const addedSeq = added?.seq;
  const addedId = added?.variantId;
  useEffect(() => {
    if (addedId)
      lines.current.get(addedId)?.scrollIntoView?.({ block: "nearest" });
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

  const backToSearch = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key !== "Enter") return;
    event.preventDefault();
    onDone();
  };

  return (
    <ol className={styles.root} aria-label="Productos de la venta">
      {totals.lines.map(({ line, amounts, quantityError, discountError }) => {
        const { item } = line;
        const name = variantName(item.name, item.label);
        const forProduct = <span className="srOnly">de {name}</span>;
        const noStock = item.variantId === outOfStock;
        const isAdded = item.variantId === added?.variantId;
        return (
          <li
            key={item.variantId}
            ref={(element) => {
              if (element) lines.current.set(item.variantId, element);
              else lines.current.delete(item.variantId);
            }}
            className={styles.line}
            aria-label={name}
            data-state={noStock ? "error" : isAdded ? "added" : undefined}
          >
            <div className={styles.product}>
              <span className={styles.name}>{name}</span>
              <span className={styles.meta}>
                {item.sku} · {formatMoney(item.priceWithTax)} / {item.unit}
              </span>
              {noStock && (
                <Badge
                  tone="danger"
                  icon={<IconAlert />}
                  className={styles.badge}
                >
                  Sin stock suficiente
                </Badge>
              )}
            </div>
            <Field
              className={styles.quantity}
              label={<>Cantidad {forProduct}</>}
              error={quantityError}
            >
              <Input
                align="end"
                inputMode="decimal"
                autoComplete="off"
                value={line.quantity}
                onChange={(event) =>
                  dispatch({
                    type: "setQuantity",
                    variantId: item.variantId,
                    text: event.target.value,
                  })
                }
                onKeyDown={backToSearch}
              />
            </Field>
            {showDiscount && (
              <Field
                className={styles.discount}
                label={
                  <>
                    <span className={styles.discountLabel}>Descuento %</span>{" "}
                    {forProduct}
                  </>
                }
                error={discountError}
              >
                <Input
                  align="end"
                  inputMode="decimal"
                  autoComplete="off"
                  placeholder="0"
                  value={line.discount}
                  onChange={(event) =>
                    dispatch({
                      type: "setDiscount",
                      variantId: item.variantId,
                      text: event.target.value,
                    })
                  }
                  onKeyDown={backToSearch}
                />
              </Field>
            )}
            <div className={styles.amount}>
              {amounts ? (
                <span className={styles.total}>
                  {formatMoney(amounts.total)}
                </span>
              ) : (
                <span className={styles.meta}>Sin total</span>
              )}
              {amounts && !isZero(amounts.discount) && (
                <span className={styles.meta}>
                  −{formatMoney(amounts.discount)} antes de IVA
                </span>
              )}
            </div>
            <IconButton
              className={styles.remove}
              label={`Quitar ${name}`}
              icon={<IconClose />}
              onClick={() => {
                dispatch({ type: "remove", variantId: item.variantId });
                onDone();
              }}
            />
          </li>
        );
      })}
    </ol>
  );
}
