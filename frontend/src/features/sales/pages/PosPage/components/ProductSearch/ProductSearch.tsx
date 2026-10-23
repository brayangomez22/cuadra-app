import { useId, type KeyboardEvent, type Ref } from 'react';
import { Field, Input } from '@/shared/ui';
import { usePosSearch, type OnAdd, type PosSearch } from '../../hooks/usePosSearch';
import { SearchResults } from '../SearchResults/SearchResults';
import styles from './ProductSearch.module.scss';

type Props = {
  onAdd: OnAdd;
  ref?: Ref<HTMLInputElement>;
};

/**
 * The point of sale's search, a combobox that keeps the focus: a barcode
 * reader types the code and Enter, and the variant with that exact SKU or
 * barcode (or a single result) goes straight to the sale. The results are
 * variants: a product with variants shows one option for each. With several,
 * the arrows choose and Enter adds. Escape clears the text.
 */
export function ProductSearch({ onAdd, ref }: Props) {
  const listId = useId();
  const search = usePosSearch(onAdd);
  const { text, term, products, results, active } = search;
  const open = Boolean(term) && (results === undefined || results.length > 0 || products.isError);
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
          onChange={(event) => search.edit(event.target.value)}
          onKeyDown={(event) => searchKeys[event.key]?.(event, search)}
          autoComplete="off"
          spellCheck={false}
          enterKeyHint="search"
          autoFocus
        />
      </Field>
      <SearchResults listId={listId} search={search} />
    </div>
  );
}

type KeyHandler = (event: KeyboardEvent<HTMLInputElement>, search: PosSearch) => void;

function moveBy(step: number): KeyHandler {
  return (event, search) => {
    const count = search.results?.length ?? 0;
    if (count === 0) return;
    event.preventDefault();
    search.setActive((i) => Math.min(Math.max(i + step, 0), count - 1));
  };
}

const searchKeys: Record<string, KeyHandler | undefined> = {
  ArrowDown: moveBy(1),
  ArrowUp: moveBy(-1),
  Enter: (event, search) => {
    event.preventDefault();
    const chosen = search.active >= 0 ? search.results?.[search.active] : undefined;
    if (chosen) search.add(chosen);
    else void search.searchNow();
  },
  Escape: (event, search) => {
    if (!search.text && !search.notFound) return;
    // Only the search: the screen's Escape (cancel the sale) waits for an empty search.
    event.preventDefault();
    event.stopPropagation();
    search.edit('');
  },
};
