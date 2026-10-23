import { describeError } from '@/shared/api/errors';
import { formatMoney } from '@/shared/format';
import { Spinner } from '@/shared/ui';
import { variantName, type Sellable } from '../../model/variants';
import styles from './PickerResults.module.scss';

type Props = {
  term: string;
  products: { isError: boolean; error: unknown };
  /** The options found; undefined while they load. */
  results: Sellable[] | undefined;
  onPick: (item: Sellable) => void;
};

/** What ProductPicker found: one option per sellable variant, or why there are none. */
export function PickerResults({ term, products, results, onPick }: Props) {
  return (
    <div className={styles.root} aria-live="polite">
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
