import { useQueryClient } from '@tanstack/react-query';
import { useId, useState, type KeyboardEvent, type Ref } from 'react';
import {
  catalogKeys,
  fetchProducts,
  findByCode,
  sellables,
  useProducts,
  variantName,
  type Product,
  type ProductQuery,
  type Sellable,
} from '@/features/catalog';
import { formatMoney, isNegative, isZero, parseDecimalInput } from '@/shared/format';
import { useDebouncedValue } from '@/shared/hooks/useDebouncedValue';
import { Field, Input, Spinner } from '@/shared/ui';
import styles from './ProductSearch.module.scss';

type Props = {
  /** Adds the variant to the sale; quantity is a decimal of the API ("1", "2.5"). */
  onAdd: (sellable: Sellable, quantity: string) => void;
  ref?: Ref<HTMLInputElement>;
};

const RESULTS = 8;

const searchQuery = (q: string): ProductQuery => ({ q, status: 'active', limit: RESULTS });

// "3*7701234567890" or "2,5*cinta": the quantity goes before the code.
const withQuantity = /^\s*(\d+(?:[.,]\d{1,4})?)\s*\*\s*(.*)$/;

function readSearch(text: string): { quantity: string; term: string } {
  const match = withQuantity.exec(text);
  if (match) {
    const parsed = parseDecimalInput(match[1] ?? '');
    if (parsed.ok && !isZero(parsed.value) && !isNegative(parsed.value)) {
      return { quantity: parsed.value, term: (match[2] ?? '').trim() };
    }
  }
  return { quantity: '1', term: text.trim() };
}

// isInactiveCode reports whether term is the code of a variant that cannot be
// sold: the search finds its product by it, but that variant is not offered.
function isInactiveCode(products: Product[], term: string) {
  const lower = term.toLowerCase();
  return products.some((p) => p.variants.some((v) => v.barcode === term || v.sku.toLowerCase() === lower));
}

/**
 * The point of sale's search, a combobox that keeps the focus: a barcode
 * reader types the code and Enter, and the variant with that exact SKU or
 * barcode (or a single result) goes straight to the sale. The results are
 * variants: a product with variants shows one option for each. With several,
 * the arrows choose and Enter adds. Escape clears the text.
 */
export function ProductSearch({ onAdd, ref }: Props) {
  const queryClient = useQueryClient();
  const listId = useId();
  const [text, setText] = useState('');
  const [active, setActive] = useState(-1);
  // Enter searches at once, without waiting for the debounce.
  const [submitted, setSubmitted] = useState<string | null>(null);
  const [notFound, setNotFound] = useState<string | null>(null);
  const { quantity, term: typed } = readSearch(text);
  const debounced = useDebouncedValue(typed);
  const term = submitted ?? debounced;
  const products = useProducts(searchQuery(term), { enabled: term !== '' });
  const page = term && !products.isPlaceholderData ? products.data?.items : undefined;
  const results = page && sellables(page);
  const open = Boolean(term) && (results === undefined || results.length > 0 || products.isError);

  function reset() {
    setText('');
    setSubmitted(null);
    setActive(-1);
  }

  function add(sellable: Sellable) {
    onAdd(sellable, quantity);
    setNotFound(null);
    reset();
  }

  async function searchNow() {
    if (!typed) return;
    setSubmitted(typed);
    try {
      const query = searchQuery(typed);
      const page = await queryClient.fetchQuery({
        queryKey: catalogKeys.products(query),
        queryFn: () => fetchProducts(query),
      });
      const options = sellables(page.items);
      const [only] = options;
      const exact = findByCode(page.items, typed);
      const pick = exact ?? (options.length === 1 ? only : undefined);
      if (pick) {
        add(pick);
      } else if (options.length === 0 || isInactiveCode(page.items, typed)) {
        // Clear the text so the next scan does not append to this code.
        setNotFound(typed);
        reset();
      } else {
        setActive(0);
      }
    } catch {
      // The results area shows the failure of the same query.
    }
  }

  function onKeyDown(event: KeyboardEvent<HTMLInputElement>) {
    const count = results?.length ?? 0;
    switch (event.key) {
      case 'ArrowDown':
        if (count === 0) return;
        event.preventDefault();
        setActive((i) => Math.min(i + 1, count - 1));
        return;
      case 'ArrowUp':
        if (count === 0) return;
        event.preventDefault();
        setActive((i) => Math.max(i - 1, 0));
        return;
      case 'Enter': {
        event.preventDefault();
        const chosen = active >= 0 ? results?.[active] : undefined;
        if (chosen) add(chosen);
        else void searchNow();
        return;
      }
      case 'Escape':
        if (!text && !notFound) return;
        // Only the search: the screen's Escape (cancel the sale) waits for an empty search.
        event.preventDefault();
        event.stopPropagation();
        setNotFound(null);
        reset();
        return;
    }
  }

  const activeId = active >= 0 && results?.[active] ? `${listId}-${active}` : undefined;

  return (
    <div className={styles.root}>
      <Field
        label="Buscar producto"
        hint={
          <>
            Escanee o escriba nombre, SKU o código. Para varias unidades: <kbd>3*código</kbd>.
          </>
        }
      >
        <Input
          ref={ref}
          role="combobox"
          aria-expanded={open}
          aria-controls={listId}
          aria-autocomplete="list"
          aria-activedescendant={activeId}
          aria-keyshortcuts="F2"
          className={styles.input}
          value={text}
          onChange={(event) => {
            setText(event.target.value);
            setSubmitted(null);
            setActive(-1);
            setNotFound(null);
          }}
          onKeyDown={onKeyDown}
          autoComplete="off"
          spellCheck={false}
          enterKeyHint="search"
          autoFocus
        />
      </Field>
      <div className={styles.results} aria-live="polite">
        {notFound ? (
          <p className={styles.message}>No encontramos «{notFound}» entre los productos activos.</p>
        ) : !term ? null : products.isError ? (
          <p className={styles.message}>No pudimos buscar los productos. {products.error.message}</p>
        ) : !results ? (
          <Spinner size="sm" label="Buscando…" />
        ) : results.length === 0 ? (
          <p className={styles.message}>No encontramos «{term}» entre los productos activos.</p>
        ) : null}
        {/* The listbox stays in the DOM: aria-controls points to it. */}
        <ul id={listId} role="listbox" aria-label="Productos encontrados" className={styles.list} hidden={!results?.length}>
          {results?.map(({ product, variant }, index) => (
            <li
              key={variant.id}
              id={`${listId}-${index}`}
              role="option"
              aria-selected={index === active}
              className={styles.option}
              // Keep the focus in the search: the next scan goes there.
              onMouseDown={(event) => event.preventDefault()}
              onClick={() => add({ product, variant })}
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
    </div>
  );
}
