import { Alert, Button, DialogClose, IconPlus } from '@/shared/ui';
import { amountText } from '../../../../model/payments';
import type { PaymentInput } from '../../../../model/paymentMethods';
import { MAX_PAYMENTS, usePayments } from '../../hooks/usePayments';
import { PaymentFigures } from '../PaymentFigures/PaymentFigures';
import { PaymentRow } from '../PaymentRow/PaymentRow';
import styles from './PaymentsForm.module.scss';

type Props = {
  /** The total of the cart, IVA included. */
  total: string;
  /** Sends the sale; the payments are already checked against the total. */
  onConfirm: (payments: PaymentInput[]) => void;
  pending: boolean;
  /** Why the API rejected the sale, in Spanish. */
  error?: string;
};

/** The payments of the sale, checked against the total before they are sent. */
export function PaymentsForm({ total, onConfirm, pending, error }: Props) {
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
      <PaymentFigures total={total} settlement={payments.settlement} />
      <fieldset className={styles.payments}>
        <legend className="srOnly">Pagos</legend>
        {rows.map((row, index) => (
          <PaymentRow
            key={row.key}
            row={row}
            index={index}
            field={{ placeholder: rows.length === 1 ? amountText(total) : undefined, error: payments.errorOf(index) }}
            onChange={(changes) => payments.change(row.key, changes)}
            onRemove={() => payments.remove(row.key)}
          />
        ))}
        {rows.length < MAX_PAYMENTS && (
          <Button variant="ghost" icon={<IconPlus />} className={styles.add} onClick={payments.add}>
            Agregar medio de pago
          </Button>
        )}
      </fieldset>
      {problem && <Alert tone="danger">{problem}</Alert>}
      {error && !problem && <Alert tone="danger">{error}</Alert>}
      <div className={styles.footer}>
        <DialogClose asChild>
          <Button variant="secondary">Volver</Button>
        </DialogClose>
        <Button type="submit" size="lg" loading={pending}>
          Confirmar venta
        </Button>
      </div>
    </form>
  );
}
