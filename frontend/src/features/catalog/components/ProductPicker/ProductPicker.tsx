import { useRef } from 'react';
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
};

/**
 * Finds an active product by name, SKU or barcode and picks one of its
 * variants: a product with variants offers one option for each. A barcode
 * reader types the code and Enter: Enter picks the variant with that exact
 * code, or the only option.
 */
export function ProductPicker({ label, value, onChange, error, onPicked }: Props) {
  const searchInput = useRef<HTMLInputElement>(null);
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
          onKeyDown={(event) => {
            if (event.key !== 'Enter') return;
            event.preventDefault(); // Enter belongs to the search, not to the form.
            void search.submit();
          }}
          autoComplete="off"
          spellCheck={false}
          enterKeyHint="search"
        />
      </Field>
      {search.term && (
        <PickerResults term={search.term} products={search.products} results={search.results} onPick={search.pick} />
      )}
    </div>
  );
}
