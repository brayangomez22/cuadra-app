import { useState } from 'react';
import { useCan } from '@/features/auth';
import { Button } from '@/shared/ui';
import type { ChosenCustomer, Customer } from '../../api/types';
import { chosenCustomer } from '../../model/chosenCustomer';
import { CustomerSearchDialog } from '../CustomerSearchDialog/CustomerSearchDialog';
import { QuickCustomerDialog } from '../QuickCustomerDialog/QuickCustomerDialog';
import styles from './CustomerPicker.module.scss';

type Props = {
  /** The customer of the sale; null sells to a final consumer. */
  value: ChosenCustomer | null;
  onChange: (customer: ChosenCustomer | null) => void;
  /** Runs once a dialog closes, e.g. to give the focus back to the product search. */
  onDone?: () => void;
};

type Step = { name: 'idle' } | { name: 'search' } | { name: 'create'; searched: string };

/**
 * Who the sale goes to: a final consumer unless the cashier picks a customer,
 * found by name or document, or creates one on the fly (customers:write).
 */
export function CustomerPicker({ value, onChange, onDone }: Props) {
  const canCreate = useCan()('customers:write');
  const [step, setStep] = useState<Step>({ name: 'idle' });
  const close = () => {
    setStep({ name: 'idle' });
    onDone?.();
  };
  const pick = (customer: Customer) => {
    onChange(chosenCustomer(customer));
    close();
  };
  return (
    <section className={styles.root} aria-label="Cliente de la venta">
      <p className={styles.customer}>
        <span className={styles.name}>{value?.name ?? 'Consumidor final'}</span>
        {value && <span className={styles.document}>{value.document}</span>}
      </p>
      <div className={styles.actions}>
        {value && (
          <Button variant="ghost" onClick={() => onChange(null)}>
            Quitar cliente
          </Button>
        )}
        <Button variant="secondary" onClick={() => setStep({ name: 'search' })}>
          {value ? 'Cambiar cliente' : 'Elegir cliente'}
        </Button>
      </div>
      <CustomerSearchDialog
        open={step.name === 'search'}
        onOpenChange={(open) => !open && close()}
        onPick={pick}
        onCreate={canCreate ? (searched) => setStep({ name: 'create', searched }) : undefined}
      />
      {step.name === 'create' && <QuickCustomerDialog searched={step.searched} onClose={close} onCreated={pick} />}
    </section>
  );
}
