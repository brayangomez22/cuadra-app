import { Alert, Button, IconPlus } from '@/shared/ui';
import { amountText, methodsFor } from '../../../../model/payments/payments';
import type { PaymentInput } from '../../../../model/payments/paymentMethods';
import { usePayments } from '../../hooks/checkout/usePayments';
import type { SaleCredit } from '../../hooks/checkout/useSaleCredit';
import { CheckoutFooter } from '../CheckoutFooter/CheckoutFooter';
import { PaymentFigures } from '../PaymentFigures/PaymentFigures';
import { PaymentRow } from '../PaymentRow/PaymentRow';
import styles from './PaymentsForm.module.scss';

type Props = {
  /** The total of the cart, IVA included. */
  total: string;
  /** What the sale's customer can buy on credit; undefined offers no credit. */
  credit: SaleCredit;
  /** Sends the sale; the payments are already checked against the total. */
  onConfirm: (payments: PaymentInput[]) => void;
  pending: boolean;
  /** Why the API rejected the sale, in Spanish. */
  error?: string;
};

/** The payments of the sale, each method at most once, checked against the total before they are sent. */
export function PaymentsForm({ total, credit, onConfirm, pending, error }: Props) {
  const payments = usePayments({ total, onConfirm, pending });
  const { rows, problem } = payments;
  return (
    <form
      className={styles.root}
      onSubmit={(event) => {
        event.preventDefault();
        payments.submit();
      }}
      noValidate
    >
      <PaymentFigures total={total} credit={credit} />
      <fieldset className={styles.payments}>
        <legend className="srOnly">Pagos</legend>
        {rows.map((row, index) => (
          <PaymentRow
            key={row.key}
            row={row}
            index={index}
            methods={methodsFor(rows, index, Boolean(credit))}
            field={{ placeholder: rows.length === 1 ? amountText(total) : undefined, error: payments.errorOf(index) }}
            onChange={(changes) => payments.change(row.key, changes)}
            onRemove={() => payments.remove(row.key)}
          />
        ))}
        {payments.canAdd && (
          <Button variant="ghost" icon={<IconPlus />} className={styles.add} onClick={payments.add}>
            Agregar medio de pago
          </Button>
        )}
      </fieldset>
      {problem && <Alert tone="danger">{problem}</Alert>}
      {error && !problem && <Alert tone="danger">{error}</Alert>}
      <CheckoutFooter settlement={payments.settlement} pending={pending} />
    </form>
  );
}
