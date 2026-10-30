import { useImperativeHandle, useRef, type KeyboardEvent, type Ref } from 'react';
import { Field, Input } from '@/shared/ui';
import { useProductSearch } from '../../hooks/useProductSearch';
import type { Sellable } from '../../model/variants/variants';
import { ChosenProduct } from '../ChosenProduct/ChosenProduct';
import { PickerResults } from '../PickerResults/PickerResults';
import styles from './ProductPicker.module.scss';

type Props = {
  label: string;
  value: Sellable | null;
  onChange: (item: Sellable | null) => void;
  error?: string;
  /** Called after the user picks a variant, e.g. to move the focus on. */
  onPicked?: (item: Sellable) => void;
  /** Lets a form focus the search, e.g. react-hook-form's field.ref. */
  ref?: Ref<{ focus: () => void }>;
};

/**
 * Finds an active product by name, SKU or barcode and picks one of its
 * variants: a product with variants offers one option for each. A barcode
 * reader types the code and Enter: Enter picks the variant with that exact
 * code, or the only option. Down from the search goes to the options.
 */
export function ProductPicker({ label, value, onChange, error, onPicked, ref }: Props) {
  const searchInput = useRef<HTMLInputElement>(null);
  useImperativeHandle(ref, () => ({ focus: () => searchInput.current?.focus() }), []);
  const resultsList = useRef<HTMLDivElement>(null);
  const search = useProductSearch({
    enabled: !value,
    onPick: (item) => {
      onChange(item);
      onPicked?.(item);
    },
  });

  if (value) {
    const change = () => {
      onChange(null);
      requestAnimationFrame(() => searchInput.current?.focus());
    };
    return <ChosenProduct label={label} value={value} onChange={change} error={error} />;
  }

  return (
    <div className={styles.root}>
      <Field label={label} error={error} hint="Nombre, SKU o código de barras. Con el lector, escanee y listo.">
        <Input
          ref={searchInput}
          type="search"
          value={search.text}
          onChange={(event) => search.change(event.target.value)}
          onKeyDown={(event) => onSearchKey(event, search.submit, resultsList.current)}
          autoComplete="off"
          spellCheck={false}
          enterKeyHint="search"
        />
      </Field>
      {search.term && (
        <PickerResults
          ref={resultsList}
          term={search.term}
          products={search.products}
          results={search.results}
          onPick={search.pick}
          onLeave={() => searchInput.current?.focus()}
        />
      )}
    </div>
  );
}

// onSearchKey: down goes to the first option; Enter searches now (a barcode
// reader ends with it) instead of submitting the form.
function onSearchKey(event: KeyboardEvent, submit: () => Promise<void>, results: HTMLElement | null) {
  if (event.key === 'ArrowDown') {
    event.preventDefault();
    results?.querySelector('button')?.focus();
  }
  if (event.key !== 'Enter') return;
  event.preventDefault();
  void submit();
}
