import { Spinner } from '@/shared/ui';
import type { Customer } from '../../api/types';
import type { CustomerSearch } from '../../hooks/useCustomerSearch';
import { documentLabel } from '../../model/documentLabel';
import styles from './CustomerSearchDialog.module.scss';

type Props = { listId: string; search: CustomerSearch; onPick: (customer: Customer) => void };

/** What the search found: a message while there is nothing to choose, and the listbox of customers. */
export function CustomerResults({ listId, search, onPick }: Props) {
  const { results, active } = search;
  return (
    <div className={styles.results} aria-live="polite">
      <SearchMessage search={search} />
      {/* The listbox stays in the DOM: aria-controls points to it. */}
      <ul
        id={listId}
        role="listbox"
        aria-label="Clientes encontrados"
        className={styles.list}
        hidden={!results?.length}
      >
        {results?.map((customer, index) => (
          <li
            key={customer.id}
            id={`${listId}-${String(index)}`}
            role="option"
            aria-selected={index === active}
            className={styles.option}
            onClick={() => onPick(customer)}
          >
            <span className={styles.name}>{customer.name}</span>
            <span className={styles.meta}>
              {[documentLabel(customer), customer.trade_name].filter(Boolean).join(' · ')}
            </span>
          </li>
        ))}
      </ul>
    </div>
  );
}

function SearchMessage({ search: { term, results, failed } }: { search: CustomerSearch }) {
  if (!term) return null;
  if (failed) return <p className={styles.message}>No pudimos buscar los clientes. Intente de nuevo.</p>;
  if (!results) return <Spinner size="sm" label="Buscando…" />;
  if (results.length === 0)
    return <p className={styles.message}>No encontramos «{term}» entre los clientes activos.</p>;
  return null;
}
