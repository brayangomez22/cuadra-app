import { useId } from 'react';
import { Alert, Button, Field, Input } from '@/shared/ui';
import type { CustomerCredit } from '../../../../api/types';
import { useCreditTermsForm } from '../../hooks/useCreditTermsForm';
import styles from './CreditTermsForm.module.scss';

type Props = { credit: CustomerCredit };

/** The credit limit and term of the customer, for who manages credit (receivables:manage). */
export function CreditTermsForm({ credit }: Props) {
  const titleId = useId();
  const terms = useCreditTermsForm(credit);
  const { errors } = terms.form.formState;
  return (
    <form className={styles.root} aria-labelledby={titleId} noValidate onSubmit={(event) => void terms.submit(event)}>
      <h2 id={titleId} className={styles.title}>
        Cupo y plazo
      </h2>
      {terms.formError && <Alert tone="danger">{terms.formError}</Alert>}
      <div className={styles.fields}>
        <Field
          label="Cupo de crédito"
          hint="Lo máximo que puede deber. Con 0 no compra a crédito."
          error={errors.credit_limit?.message}
        >
          <Input align="end" inputMode="decimal" autoComplete="off" {...terms.form.register('credit_limit')} />
        </Field>
        <Field label="Plazo en días" hint="Para pagar cada venta a crédito." error={errors.term_days?.message}>
          <Input align="end" inputMode="numeric" autoComplete="off" {...terms.form.register('term_days')} />
        </Field>
      </div>
      <Button type="submit" variant="secondary" className={styles.save} loading={terms.isSaving}>
        Guardar cupo
      </Button>
    </form>
  );
}
