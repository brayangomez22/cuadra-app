import { variantName } from '@/features/catalog';
import { formatMoney } from '@/shared/format';
import { Spinner } from '@/shared/ui';
import type { PosSearch } from '../../hooks/usePosSearch';
import styles from './SearchResults.module.scss';

type Props = { listId: string; search: PosSearch };

/** What the search found: a message while there is nothing to choose, and the listbox of variants. */
export function SearchResults({ listId, search }: Props) {
  const { results, active } = search;
  return (
    <div className={styles.root} aria-live="polite">
      <SearchMessage search={search} />
      {/* The listbox stays in the DOM: aria-controls points to it. */}
      <ul
        id={listId}
        role="listbox"
        aria-label="Productos encontrados"
        className={styles.list}
        hidden={!results?.length}
      >
        {results?.map(({ product, variant }, index) => (
          <li
            key={variant.id}
            id={`${listId}-${index}`}
            role="option"
            aria-selected={index === active}
            className={styles.option}
            // Keep the focus in the search: the next scan goes there.
            onMouseDown={(event) => event.preventDefault()}
            onClick={() => search.add({ product, variant })}
          >
            <span className={styles.name}>{variantName(product.name, variant.label)}</span>
            <span className={styles.meta}>
              {variant.sku} · {product.unit.code}
            </span>
            <span className={styles.price}>{formatMoney(variant.price_with_tax)}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}

function SearchMessage({ search: { notFound, term, products, results } }: { search: PosSearch }) {
  if (notFound) return <p className={styles.message}>No encontramos «{notFound}» entre los productos activos.</p>;
  if (!term) return null;
  if (products.isError) {
    return <p className={styles.message}>No pudimos buscar los productos. {products.error.message}</p>;
  }
  if (!results) return <Spinner size="sm" label="Buscando…" />;
  if (results.length === 0) {
    return <p className={styles.message}>No encontramos «{term}» entre los productos activos.</p>;
  }
  return null;
}
