import { useState } from 'react';
import { useCan } from '@/features/auth';
import type { ChosenCustomer, Customer } from '../../api/types';
import { chosenCustomer } from '../../model/chosenCustomer';
import { CustomerChoice } from './CustomerChoice';
import { CustomerSearchDialog } from '../CustomerSearchDialog/CustomerSearchDialog';
import { QuickCustomerDialog } from '../QuickCustomerDialog/QuickCustomerDialog';
import styles from './CustomerPicker.module.scss';

type Props = {
  /** The customer of the sale; null sells to a final consumer. */
  value: ChosenCustomer | null;
  onChange: (customer: ChosenCustomer | null) => void;
  /** Whether the customer is being chosen (searched or created): the screen opens it too, e.g. with a shortcut. */
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** The key that opens it, announced on its button (aria-keyshortcuts). */
  shortcut?: string;
  /** Runs once a dialog closes, e.g. to give the focus back to the product search. */
  onDone?: () => void;
};

/**
 * Who the sale goes to: a final consumer unless the cashier picks a customer,
 * found by name or document, or creates one on the fly (customers:write).
 */
export function CustomerPicker({ value, onChange, open, onOpenChange, shortcut, onDone }: Props) {
  const canCreate = useCan()('customers:write');
  // What was searched when the cashier chose to create the customer.
  const [creating, setCreating] = useState<string>();
  const close = () => {
    setCreating(undefined);
    onOpenChange(false);
    onDone?.();
  };
  const pick = (customer: Customer) => {
    onChange(chosenCustomer(customer));
    close();
  };
  return (
    <section className={styles.root} aria-label="Cliente de la venta">
      <CustomerChoice
        value={value}
        onClear={() => onChange(null)}
        onChoose={() => onOpenChange(true)}
        shortcut={shortcut}
      />
      <CustomerSearchDialog
        open={open && creating === undefined}
        onOpenChange={(next) => !next && close()}
        onPick={pick}
        onCreate={canCreate ? setCreating : undefined}
      />
      {open && creating !== undefined && <QuickCustomerDialog searched={creating} onClose={close} onCreated={pick} />}
    </section>
  );
}
