import { formatMoney, formatQuantity, isZero } from '@/shared/format';
import type { Sale } from '../../api/types';
import styles from './ReceiptLines.module.scss';

/** The lines of the receipt: the name, quantity × price, the amount and the discount. */
export function ReceiptLines({ lines }: { lines: Sale['lines'] }) {
  return (
    <ul className={styles.root}>
      {lines.map((line, index) => (
        <li key={`${line.variant_id}-${index}`} className={styles.line} aria-label={lineName(line)}>
          <span className={styles.name}>{lineName(line)}</span>
          <span>
            {formatQuantity(line.quantity)}
            {line.unit_code && ` ${line.unit_code}`} × {formatMoney(line.unit_price)}
          </span>
          <span className={styles.amount}>{formatMoney(line.gross)}</span>
          {!isZero(line.discount) && (
            <>
              <span>Descuento</span>
              <span className={styles.amount}>{formatMoney(`-${line.discount}`)}</span>
            </>
          )}
        </li>
      ))}
    </ul>
  );
}

// lineName is the product of a line and, for a variant with axes, its label
// at the time of the sale: "Producto A (Rojo · M)".
function lineName(line: Sale['lines'][number]) {
  return line.variant_label ? `${line.product_name} (${line.variant_label})` : line.product_name;
}
