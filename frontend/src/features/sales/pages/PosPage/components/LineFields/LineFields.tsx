import type { Dispatch, KeyboardEvent, ReactNode } from 'react';
import { Field, Input } from '@/shared/ui';
import type { LineRef } from '../../../../model/cart/cartLine';
import type { CartAction } from '../../../../model/cart/cartReducer';
import type { LineTotals } from '../../../../model/cart/lineTotals';
import styles from './LineFields.module.scss';

type FieldsProps = {
  totals: LineTotals;
  lineRef: LineRef;
  dispatch: Dispatch<CartAction>;
  /** Completes each label for a screen reader: "de Tubo PVC". */
  forProduct: ReactNode;
  showDiscount: boolean;
  onDone: () => void;
};

/** The quantity and the discount of a line, edited in place; Enter in either goes back to the search. */
export function LineFields({
  totals: { line, quantityError, discountError },
  lineRef,
  dispatch,
  forProduct,
  showDiscount,
  onDone,
}: FieldsProps) {
  const backToSearch = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key !== 'Enter') return;
    event.preventDefault();
    onDone();
  };
  const input = { align: 'end', inputMode: 'decimal', autoComplete: 'off', onKeyDown: backToSearch } as const;
  return (
    <>
      <Field className={styles.quantity} label={<>Cantidad {forProduct}</>} error={quantityError}>
        <Input
          {...input}
          value={line.quantity}
          onChange={(event) => dispatch({ type: 'setQuantity', ...lineRef, text: event.target.value })}
        />
      </Field>
      {showDiscount && (
        <Field
          className={styles.discount}
          label={
            <>
              <span className={styles.discountLabel}>Descuento %</span> {forProduct}
            </>
          }
          error={discountError}
        >
          <Input
            {...input}
            placeholder="0"
            value={line.discount}
            onChange={(event) => dispatch({ type: 'setDiscount', ...lineRef, text: event.target.value })}
          />
        </Field>
      )}
    </>
  );
}
