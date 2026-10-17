import { useQueryClient } from '@tanstack/react-query';
import { useRef, useState, type KeyboardEvent } from 'react';
import { formatMoney } from '@/shared/format';
import { useDebouncedValue } from '@/shared/hooks/useDebouncedValue';
import { Button, Field, Input, Spinner } from '@/shared/ui';
import { catalogKeys, fetchProducts, useProducts, type ProductQuery } from './api';
import { findByCode, sellables, variantName, type Sellable } from './variants';
import styles from './ProductPicker.module.scss';

type Props = {
  label: string;
  value: Sellable | null;
  onChange: (item: Sellable | null) => void;
  error?: string;
  /** Called after the user picks a variant, e.g. to move the focus on. */
  onPicked?: (item: Sellable) => void;
};

const RESULTS = 8;

const searchQuery = (q: string): ProductQuery => ({ q, status: 'active', limit: RESULTS });

/**
 * Finds an active product by name, SKU or barcode and picks one of its
 * variants: a product with variants offers one option for each. A barcode
 * reader types the code and Enter: Enter picks the variant with that exact
 * code, or the only option.
 */
export function ProductPicker({ label, value, onChange, error, onPicked }: Props) {
  const queryClient = useQueryClient();
  const [text, setText] = useState('');
  const typed = useDebouncedValue(text.trim());
  // Enter searches at once, without waiting for the debounce.
  const [submitted, setSubmitted] = useState<string | null>(null);
  const term = submitted ?? typed;
  const search = useRef<HTMLInputElement>(null);
  const products = useProducts(searchQuery(term), { enabled: term !== '' && !value });
  const page = term && !products.isPlaceholderData ? products.data?.items : undefined;
  const results = page && sellables(page);

  function pick(item: Sellable) {
    onChange(item);
    setText('');
    setSubmitted(null);
    onPicked?.(item);
  }

  async function onKeyDown(event: KeyboardEvent<HTMLInputElement>) {
    if (event.key !== 'Enter') return;
    event.preventDefault(); // Enter belongs to the search, not to the form.
    const q = text.trim();
    if (!q) return;
    setSubmitted(q);
    try {
      const query = searchQuery(q);
      const page = await queryClient.fetchQuery({
        queryKey: catalogKeys.products(query),
        queryFn: () => fetchProducts(query),
      });
      const options = sellables(page.items);
      const [only] = options;
      const chosen = findByCode(page.items, q) ?? (options.length === 1 ? only : undefined);
      if (chosen) pick(chosen);
    } catch {
      // The results area shows the failure of the same query.
    }
  }

  if (value) {
    return (
      <div className={styles.root}>
        <span className={styles.label}>{label}</span>
        <div className={styles.chosen} role="group" aria-label={`${label} elegido`}>
          <div className={styles.chosenText}>
            <span className={styles.name}>{variantName(value.product.name, value.variant.label)}</span>
            <span className={styles.meta}>
              {value.variant.sku} · {value.product.unit.code}
            </span>
          </div>
          <Button
            variant="secondary"
            onClick={() => {
              onChange(null);
              requestAnimationFrame(() => search.current?.focus());
            }}
          >
            Cambiar
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className={styles.root}>
      <Field label={label} error={error} hint="Nombre, SKU o código de barras. Con el lector, escanee y listo.">
        <Input
          ref={search}
          type="search"
          value={text}
          onChange={(event) => {
            setText(event.target.value);
            setSubmitted(null);
          }}
          onKeyDown={(event) => void onKeyDown(event)}
          autoComplete="off"
          spellCheck={false}
          enterKeyHint="search"
        />
      </Field>
      {term && (
        <div className={styles.results} aria-live="polite">
          {products.isError ? (
            <p className={styles.empty}>No pudimos buscar los productos. {products.error.message}</p>
          ) : !results ? (
            <Spinner size="sm" label="Buscando…" />
          ) : results.length === 0 ? (
            <p className={styles.empty}>No encontramos «{term}» entre los productos activos.</p>
          ) : (
            <ul className={styles.list} aria-label="Resultados">
              {results.map((item) => (
                <li key={item.variant.id} className={styles.item}>
                  <button type="button" className={styles.option} onClick={() => pick(item)}>
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
      )}
    </div>
  );
}
