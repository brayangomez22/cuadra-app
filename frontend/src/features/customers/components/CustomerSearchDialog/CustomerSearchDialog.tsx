import { useId } from 'react';
import { Button, Dialog, IconPlus, Input } from '@/shared/ui';
import type { Customer } from '../../api/types';
import { useCustomerSearch } from '../../hooks/useCustomerSearch';
import { CustomerResults } from './CustomerResults';
import styles from './CustomerSearchDialog.module.scss';

type Props = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onPick: (customer: Customer) => void;
  /** Opens the quick creation with what was searched; without it, the user cannot create customers. */
  onCreate?: (searched: string) => void;
};

/** Finds an active customer by name or document; Enter picks the highlighted one. */
export function CustomerSearchDialog({ open, onOpenChange, onPick, onCreate }: Props) {
  const search = useCustomerSearch();
  const listId = useId();
  const picked = search.results?.[search.active];
  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title="Elegir cliente"
      description="Busque por el nombre, el nombre comercial o el número de documento."
      size="md"
    >
      <div className={styles.root}>
        <Input
          role="combobox"
          aria-label="Buscar cliente"
          aria-expanded={Boolean(search.results?.length)}
          aria-controls={listId}
          aria-activedescendant={picked ? `${listId}-${String(search.active)}` : undefined}
          aria-autocomplete="list"
          type="search"
          value={search.text}
          onChange={(event) => search.change(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
              event.preventDefault();
              search.move(event.key === 'ArrowDown' ? 1 : -1);
            } else if (event.key === 'Enter' && picked) {
              event.preventDefault();
              onPick(picked);
            }
          }}
          autoComplete="off"
          spellCheck={false}
          enterKeyHint="search"
          autoFocus
        />
        <CustomerResults listId={listId} search={search} onPick={onPick} />
        {onCreate && (
          <Button variant="secondary" icon={<IconPlus />} onClick={() => onCreate(search.text)}>
            Nuevo cliente
          </Button>
        )}
      </div>
    </Dialog>
  );
}
