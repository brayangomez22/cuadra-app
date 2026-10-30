import type { KeyboardEvent, Ref } from 'react';
import { describeError } from '@/shared/api/errors';
import { formatMoney } from '@/shared/format';
import { Spinner } from '@/shared/ui';
import { variantName, type Sellable } from '../../model/variants/variants';
import styles from './PickerResults.module.scss';

type Props = {
  term: string;
  products: { isError: boolean; error: unknown };
  /** The options found; undefined while they load. */
  results: Sellable[] | undefined;
  onPick: (item: Sellable) => void;
  /** Up from the first option: back to the search. */
  onLeave: () => void;
  ref?: Ref<HTMLDivElement>;
};

/**
 * What ProductPicker found: one option per sellable variant, or why there
 * are none. The arrows move between the options; Enter picks one.
 */
export function PickerResults({ term, products, results, onPick, onLeave, ref }: Props) {
  return (
    <div className={styles.root} aria-live="polite" ref={ref} onKeyDown={(event) => moveFocus(event, onLeave)}>
      {products.isError ? (
        <p className={styles.empty}>No pudimos buscar los productos. {describeError(products.error).message}</p>
      ) : !results ? (
        <Spinner size="sm" label="Buscando…" />
      ) : results.length === 0 ? (
        <p className={styles.empty}>No encontramos «{term}» entre los productos activos.</p>
      ) : (
        <ul className={styles.list} aria-label="Resultados">
          {results.map((item) => (
            <li key={item.variant.id} className={styles.item}>
              <button type="button" className={styles.option} onClick={() => onPick(item)}>
                <span className={styles.name}>{variantName(item.product.name, item.variant.label)}</span>
                <span className={styles.meta}>
                  {item.variant.sku} · {item.product.unit.code} · {formatMoney(item.variant.price_with_tax)}
                </span>
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

// moveFocus takes ↑ and ↓ from an option to the one before or after it; up
// from the first leaves the list.
function moveFocus(event: KeyboardEvent<HTMLDivElement>, onLeave: () => void) {
  if (event.key !== 'ArrowDown' && event.key !== 'ArrowUp') return;
  const options = [...event.currentTarget.querySelectorAll('button')];
  const at = options.indexOf(event.target as HTMLButtonElement);
  if (at < 0) return;
  event.preventDefault();
  const next = at + (event.key === 'ArrowDown' ? 1 : -1);
  if (next < 0) onLeave();
  else options[Math.min(next, options.length - 1)]?.focus();
}
